package monitoring

import (
	"context"
	"errors"
	"time"

	"github.com/740297616/nexusops/server/internal/serverasset"
	"gorm.io/gorm"
)

var ErrDuplicate = errors.New("duplicate metric request")
var ErrInvalid = errors.New("invalid metric request")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Insert(ctx context.Context, principal serverasset.AgentPrincipal, req IngestRequest, received time.Time) (int, error) {
	returned := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sampleID int64
		result := tx.Raw(`INSERT INTO metric_samples(server_id,agent_credential_id,request_id,schema_version,sampled_at,received_at) VALUES(?,?,?,?,?,?) ON CONFLICT (server_id,request_id) DO NOTHING RETURNING id`, principal.ServerID, principal.CredentialID, req.RequestID, req.SchemaVersion, req.SampledAt, received).Scan(&sampleID)
		if result.Error != nil {
			return result.Error
		}
		if sampleID == 0 {
			return ErrDuplicate
		}
		for _, item := range req.Samples {
			labels := item.Labels
			switch item.MetricType {
			case "cpu", "memory":
				if err := tx.Exec("INSERT INTO metric_values(sample_id,metric_type,metric_name,value,unit) VALUES(?,?,?,?,?)", sampleID, item.MetricType, item.MetricName, item.Value, item.Unit).Error; err != nil {
					return err
				}
			case "filesystem":
				mount := labels["mountpoint"]
				fs := labels["filesystem"]
				if mount == "" {
					return ErrInvalid
				}
				if err := tx.Exec("INSERT INTO metric_filesystems(sample_id,mountpoint,filesystem,metric_name,value,unit) VALUES(?,?,?,?,?,?)", sampleID, mount, fs, item.MetricName, item.Value, item.Unit).Error; err != nil {
					return err
				}
			case "network":
				iface := labels["interface"]
				if iface == "" {
					return ErrInvalid
				}
				if err := tx.Exec("INSERT INTO metric_networks(sample_id,interface_name,metric_name,value,unit) VALUES(?,?,?,?,?)", sampleID, iface, item.MetricName, item.Value, item.Unit).Error; err != nil {
					return err
				}
			default:
				return ErrInvalid
			}
			returned++
		}
		return nil
	})
	return returned, err
}

func (r *Repository) Latest(ctx context.Context, serverID string, now time.Time, staleAfter time.Duration) ([]Point, error) {
	var rows []Point
	err := r.db.WithContext(ctx).Raw(`SELECT v.metric_type, v.metric_name, '{}'::jsonb AS labels, v.value, v.unit, s.sampled_at, s.received_at FROM metric_values v JOIN metric_samples s ON s.id=v.sample_id WHERE s.server_id=? ORDER BY s.sampled_at DESC LIMIT 100`, serverID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	var fs []struct {
		MetricName, Mountpoint, Filesystem, Unit string
		Value                                    float64
		SampledAt, ReceivedAt                    time.Time
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT f.metric_name,f.mountpoint,f.filesystem,f.unit,f.value,s.sampled_at,s.received_at FROM metric_filesystems f JOIN metric_samples s ON s.id=f.sample_id WHERE s.server_id=? ORDER BY s.sampled_at DESC LIMIT 200`, serverID).Scan(&fs).Error; err != nil {
		return nil, err
	}
	for _, row := range fs {
		rows = append(rows, Point{MetricType: "filesystem", MetricName: row.MetricName, Labels: map[string]string{"mountpoint": row.Mountpoint, "filesystem": row.Filesystem}, Value: row.Value, Unit: row.Unit, SampledAt: row.SampledAt, ReceivedAt: row.ReceivedAt})
	}
	var nw []struct {
		MetricName, InterfaceName, Unit string
		Value                           float64
		SampledAt, ReceivedAt           time.Time
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT n.metric_name,n.interface_name,n.unit,n.value,s.sampled_at,s.received_at FROM metric_networks n JOIN metric_samples s ON s.id=n.sample_id WHERE s.server_id=? ORDER BY s.sampled_at DESC LIMIT 200`, serverID).Scan(&nw).Error; err != nil {
		return nil, err
	}
	for _, row := range nw {
		rows = append(rows, Point{MetricType: "network", MetricName: row.MetricName, Labels: map[string]string{"interface": row.InterfaceName}, Value: row.Value, Unit: row.Unit, SampledAt: row.SampledAt, ReceivedAt: row.ReceivedAt})
	}
	_ = now
	_ = staleAfter
	return rows, nil
}

func (r *Repository) History(ctx context.Context, serverID, metricType, metricName, labelKey, labelValue string, from, to time.Time, limit int) ([]Point, error) {
	if metricType == "cpu" || metricType == "memory" {
		var rows []Point
		err := r.db.WithContext(ctx).Raw(`SELECT v.metric_type,v.metric_name,'{}'::jsonb AS labels,v.value,v.unit,s.sampled_at,s.received_at FROM metric_values v JOIN metric_samples s ON s.id=v.sample_id WHERE s.server_id=? AND v.metric_type=? AND v.metric_name=? AND s.sampled_at>=? AND s.sampled_at<=? ORDER BY s.sampled_at LIMIT ?`, serverID, metricType, metricName, from, to, limit).Scan(&rows).Error
		return rows, err
	}
	if metricType == "filesystem" {
		var rows []Point
		err := r.db.WithContext(ctx).Raw(`SELECT 'filesystem',f.metric_name,jsonb_build_object('mountpoint',f.mountpoint,'filesystem',f.filesystem),f.value,f.unit,s.sampled_at,s.received_at FROM metric_filesystems f JOIN metric_samples s ON s.id=f.sample_id WHERE s.server_id=? AND f.metric_name=? AND f.mountpoint=? AND s.sampled_at>=? AND s.sampled_at<=? ORDER BY s.sampled_at LIMIT ?`, serverID, metricName, labelValue, from, to, limit).Scan(&rows).Error
		return rows, err
	}
	if metricType == "network" {
		var rows []Point
		err := r.db.WithContext(ctx).Raw(`SELECT 'network',n.metric_name,jsonb_build_object('interface',n.interface_name),n.value,n.unit,s.sampled_at,s.received_at FROM metric_networks n JOIN metric_samples s ON s.id=n.sample_id WHERE s.server_id=? AND n.metric_name=? AND n.interface_name=? AND s.sampled_at>=? AND s.sampled_at<=? ORDER BY s.sampled_at LIMIT ?`, serverID, metricName, labelValue, from, to, limit).Scan(&rows).Error
		return rows, err
	}
	return nil, ErrInvalid
}

func (r *Repository) Cleanup(ctx context.Context, before time.Time, batch int) (int64, error) {
	var removed int64
	for {
		var ids []int64
		if err := r.db.WithContext(ctx).Table("metric_samples").Where("received_at < ?", before).Order("id").Limit(batch).Pluck("id", &ids).Error; err != nil {
			return removed, err
		}
		if len(ids) == 0 {
			break
		}
		result := r.db.WithContext(ctx).Exec("DELETE FROM metric_samples WHERE id IN ?", ids)
		if result.Error != nil {
			return removed, result.Error
		}
		removed += result.RowsAffected
		if len(ids) < batch {
			break
		}
	}
	return removed, nil
}
