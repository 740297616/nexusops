package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/740297616/nexusops/server/internal/serverasset"
	"gorm.io/gorm"
)

var ErrDuplicate = errors.New("duplicate metric request")
var ErrInvalid = errors.New("invalid metric request")

type Repository struct{ db *gorm.DB }

type rawPoint struct {
	MetricType string
	MetricName string
	Labels     string
	Value      float64
	Unit       string
	SampledAt  time.Time
	ReceivedAt time.Time
}

func rawPoints(rows []rawPoint) []Point {
	points := make([]Point, 0, len(rows))
	for _, row := range rows {
		labels := map[string]string{}
		if row.Labels != "" {
			_ = json.Unmarshal([]byte(row.Labels), &labels)
		}
		points = append(points, Point{MetricType: row.MetricType, MetricName: row.MetricName, Labels: labels, Value: row.Value, Unit: row.Unit, SampledAt: row.SampledAt, ReceivedAt: row.ReceivedAt})
	}
	return points
}

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
	var rows []rawPoint
	err := r.db.WithContext(ctx).Raw(`SELECT DISTINCT ON (v.metric_type,v.metric_name) v.metric_type, v.metric_name, '{}'::jsonb AS labels, v.value, v.unit, s.sampled_at, s.received_at FROM metric_values v JOIN metric_samples s ON s.id=v.sample_id WHERE s.server_id=? ORDER BY v.metric_type,v.metric_name,s.sampled_at DESC,s.received_at DESC,s.id DESC LIMIT 100`, serverID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	points := rawPoints(rows)
	var fs []struct {
		MetricName, Mountpoint, Filesystem, Unit string
		Value                                    float64
		SampledAt, ReceivedAt                    time.Time
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT DISTINCT ON (f.mountpoint,f.metric_name) f.metric_name,f.mountpoint,f.filesystem,f.unit,f.value,s.sampled_at,s.received_at FROM metric_filesystems f JOIN metric_samples s ON s.id=f.sample_id WHERE s.server_id=? ORDER BY f.mountpoint,f.metric_name,s.sampled_at DESC,s.received_at DESC,s.id DESC LIMIT 200`, serverID).Scan(&fs).Error; err != nil {
		return nil, err
	}
	for _, row := range fs {
		points = append(points, Point{MetricType: "filesystem", MetricName: row.MetricName, Labels: map[string]string{"mountpoint": row.Mountpoint, "filesystem": row.Filesystem}, Value: row.Value, Unit: row.Unit, SampledAt: row.SampledAt, ReceivedAt: row.ReceivedAt})
	}
	var nw []struct {
		MetricName, InterfaceName, Unit string
		Value                           float64
		SampledAt, ReceivedAt           time.Time
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT DISTINCT ON (n.interface_name,n.metric_name) n.metric_name,n.interface_name,n.unit,n.value,s.sampled_at,s.received_at FROM metric_networks n JOIN metric_samples s ON s.id=n.sample_id WHERE s.server_id=? ORDER BY n.interface_name,n.metric_name,s.sampled_at DESC,s.received_at DESC,s.id DESC LIMIT 200`, serverID).Scan(&nw).Error; err != nil {
		return nil, err
	}
	for _, row := range nw {
		points = append(points, Point{MetricType: "network", MetricName: row.MetricName, Labels: map[string]string{"interface": row.InterfaceName}, Value: row.Value, Unit: row.Unit, SampledAt: row.SampledAt, ReceivedAt: row.ReceivedAt})
	}
	_ = now
	_ = staleAfter
	return points, nil
}

func (r *Repository) History(ctx context.Context, serverID, metricType, metricName, labelKey, labelValue string, from, to time.Time, limit int) ([]Point, error) {
	if metricType == "cpu" || metricType == "memory" {
		var rows []rawPoint
		err := r.db.WithContext(ctx).Raw(`SELECT v.metric_type,v.metric_name,'{}'::jsonb AS labels,v.value,v.unit,s.sampled_at,s.received_at FROM metric_values v JOIN metric_samples s ON s.id=v.sample_id WHERE s.server_id=? AND v.metric_type=? AND v.metric_name=? AND s.sampled_at>=? AND s.sampled_at<=? ORDER BY s.sampled_at LIMIT ?`, serverID, metricType, metricName, from, to, limit).Scan(&rows).Error
		return rawPoints(rows), err
	}
	if metricType == "filesystem" {
		var rows []rawPoint
		err := r.db.WithContext(ctx).Raw(`SELECT 'filesystem' AS metric_type,f.metric_name,jsonb_build_object('mountpoint',f.mountpoint,'filesystem',f.filesystem)::text AS labels,f.value,f.unit,s.sampled_at,s.received_at FROM metric_filesystems f JOIN metric_samples s ON s.id=f.sample_id WHERE s.server_id=? AND f.metric_name=? AND f.mountpoint=? AND s.sampled_at>=? AND s.sampled_at<=? ORDER BY s.sampled_at LIMIT ?`, serverID, metricName, labelValue, from, to, limit).Scan(&rows).Error
		return rawPoints(rows), err
	}
	if metricType == "network" {
		var rows []rawPoint
		err := r.db.WithContext(ctx).Raw(`SELECT 'network' AS metric_type,n.metric_name,jsonb_build_object('interface',n.interface_name)::text AS labels,n.value,n.unit,s.sampled_at,s.received_at FROM metric_networks n JOIN metric_samples s ON s.id=n.sample_id WHERE s.server_id=? AND n.metric_name=? AND n.interface_name=? AND s.sampled_at>=? AND s.sampled_at<=? ORDER BY s.sampled_at LIMIT ?`, serverID, metricName, labelValue, from, to, limit).Scan(&rows).Error
		return rawPoints(rows), err
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

func (r *Repository) OldestSample(ctx context.Context) (time.Time, error) {
	var oldest sql.NullTime
	if err := r.db.WithContext(ctx).Raw("SELECT MIN(sampled_at) FROM metric_samples").Row().Scan(&oldest); err != nil {
		return time.Time{}, err
	}
	return oldest.Time, nil
}

func (r *Repository) Aggregate(ctx context.Context, from, to time.Time, stepSeconds int) error {
	if stepSeconds != 300 && stepSeconds != 3600 {
		return ErrInvalid
	}
	interval := "5 minutes"
	if stepSeconds == 3600 {
		interval = "1 hour"
	}
	queries := []string{
		`INSERT INTO metric_rollups(server_id,bucket_start,step_seconds,metric_type,metric_name,labels,avg_value,min_value,max_value,last_value,last_sampled_at,last_received_at,unit,sample_count)
SELECT s.server_id,date_bin(?::interval,s.sampled_at,'1970-01-01 UTC'::timestamptz),?::int,v.metric_type,v.metric_name,'{}'::jsonb,avg(v.value),min(v.value),max(v.value),(array_agg(v.value ORDER BY s.sampled_at DESC,s.received_at DESC,s.id DESC))[1],(array_agg(s.sampled_at ORDER BY s.sampled_at DESC,s.received_at DESC,s.id DESC))[1],(array_agg(s.received_at ORDER BY s.sampled_at DESC,s.received_at DESC,s.id DESC))[1],min(v.unit),count(*)
FROM metric_values v JOIN metric_samples s ON s.id=v.sample_id WHERE s.sampled_at>=? AND s.sampled_at<?
GROUP BY s.server_id,2,3,4,5
ON CONFLICT (server_id,bucket_start,step_seconds,metric_type,metric_name,labels) DO UPDATE SET avg_value=EXCLUDED.avg_value,min_value=EXCLUDED.min_value,max_value=EXCLUDED.max_value,last_value=EXCLUDED.last_value,last_sampled_at=EXCLUDED.last_sampled_at,last_received_at=EXCLUDED.last_received_at,unit=EXCLUDED.unit,sample_count=EXCLUDED.sample_count WHERE EXCLUDED.sample_count >= metric_rollups.sample_count`,
		`INSERT INTO metric_rollups(server_id,bucket_start,step_seconds,metric_type,metric_name,labels,avg_value,min_value,max_value,last_value,last_sampled_at,last_received_at,unit,sample_count)
SELECT s.server_id,date_bin(?::interval,s.sampled_at,'1970-01-01 UTC'::timestamptz),?::int, 'filesystem',f.metric_name,jsonb_build_object('mountpoint',f.mountpoint,'filesystem',f.filesystem),avg(f.value),min(f.value),max(f.value),(array_agg(f.value ORDER BY s.sampled_at DESC,s.received_at DESC,s.id DESC))[1],(array_agg(s.sampled_at ORDER BY s.sampled_at DESC,s.received_at DESC,s.id DESC))[1],(array_agg(s.received_at ORDER BY s.sampled_at DESC,s.received_at DESC,s.id DESC))[1],min(f.unit),count(*)
FROM metric_filesystems f JOIN metric_samples s ON s.id=f.sample_id WHERE s.sampled_at>=? AND s.sampled_at<?
GROUP BY s.server_id,2,3,4,5,6
ON CONFLICT (server_id,bucket_start,step_seconds,metric_type,metric_name,labels) DO UPDATE SET avg_value=EXCLUDED.avg_value,min_value=EXCLUDED.min_value,max_value=EXCLUDED.max_value,last_value=EXCLUDED.last_value,last_sampled_at=EXCLUDED.last_sampled_at,last_received_at=EXCLUDED.last_received_at,unit=EXCLUDED.unit,sample_count=EXCLUDED.sample_count WHERE EXCLUDED.sample_count >= metric_rollups.sample_count`,
		`INSERT INTO metric_rollups(server_id,bucket_start,step_seconds,metric_type,metric_name,labels,avg_value,min_value,max_value,last_value,last_sampled_at,last_received_at,unit,sample_count)
SELECT s.server_id,date_bin(?::interval,s.sampled_at,'1970-01-01 UTC'::timestamptz),?::int, 'network',n.metric_name,jsonb_build_object('interface',n.interface_name),avg(n.value),min(n.value),max(n.value),(array_agg(n.value ORDER BY s.sampled_at DESC,s.received_at DESC,s.id DESC))[1],(array_agg(s.sampled_at ORDER BY s.sampled_at DESC,s.received_at DESC,s.id DESC))[1],(array_agg(s.received_at ORDER BY s.sampled_at DESC,s.received_at DESC,s.id DESC))[1],min(n.unit),count(*)
FROM metric_networks n JOIN metric_samples s ON s.id=n.sample_id WHERE s.sampled_at>=? AND s.sampled_at<?
GROUP BY s.server_id,2,3,4,5,6
ON CONFLICT (server_id,bucket_start,step_seconds,metric_type,metric_name,labels) DO UPDATE SET avg_value=EXCLUDED.avg_value,min_value=EXCLUDED.min_value,max_value=EXCLUDED.max_value,last_value=EXCLUDED.last_value,last_sampled_at=EXCLUDED.last_sampled_at,last_received_at=EXCLUDED.last_received_at,unit=EXCLUDED.unit,sample_count=EXCLUDED.sample_count WHERE EXCLUDED.sample_count >= metric_rollups.sample_count`,
	}
	for _, query := range queries {
		if err := r.db.WithContext(ctx).Exec(query, interval, stepSeconds, from, to).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) RollupHistory(ctx context.Context, serverID, metricType, metricName string, labelKey, labelValue string, from, to time.Time, stepSeconds, limit int) ([]Point, error) {
	var rows []struct {
		MetricType, MetricName, Labels, Unit       string
		AvgValue, MinValue, MaxValue, LastValue    float64
		SampleCount                                int64
		BucketStart, LastSampledAt, LastReceivedAt time.Time
	}
	q := `SELECT metric_type,metric_name,labels::text AS labels,unit,avg_value,min_value,max_value,last_value,sample_count,bucket_start,last_sampled_at,last_received_at FROM metric_rollups WHERE server_id=? AND step_seconds=? AND metric_type=? AND metric_name=? AND bucket_start>=? AND bucket_start<=?`
	args := []any{serverID, stepSeconds, metricType, metricName, from, to}
	if labelKey != "" {
		q += " AND labels @> ?::jsonb"
		label, err := json.Marshal(map[string]string{labelKey: labelValue})
		if err != nil {
			return nil, err
		}
		args = append(args, string(label))
	}
	q += " ORDER BY bucket_start LIMIT ?"
	args = append(args, limit)
	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	points := make([]Point, 0, len(rows))
	for _, row := range rows {
		var labels map[string]string
		_ = json.Unmarshal([]byte(row.Labels), &labels)
		value := row.AvgValue
		if strings.HasSuffix(row.MetricName, "_total") {
			value = row.LastValue
		}
		points = append(points, Point{MetricType: row.MetricType, MetricName: row.MetricName, Labels: labels, Value: value, Unit: row.Unit, SampledAt: row.BucketStart, ReceivedAt: row.LastReceivedAt, LastSampledAt: &row.LastSampledAt, MinValue: &row.MinValue, MaxValue: &row.MaxValue, SampleCount: &row.SampleCount})
	}
	return points, nil
}

func (r *Repository) CleanupRollups(ctx context.Context, before5m, before1h time.Time, batch int) (int64, error) {
	var total int64
	for _, item := range []struct {
		step   int
		before time.Time
	}{{300, before5m}, {3600, before1h}} {
		for {
			res := r.db.WithContext(ctx).Exec(`WITH doomed AS (
SELECT ctid FROM metric_rollups WHERE step_seconds=? AND bucket_start + step_seconds * interval '1 second' <= ? ORDER BY bucket_start LIMIT ?
)
DELETE FROM metric_rollups WHERE ctid IN (SELECT ctid FROM doomed)`, item.step, item.before, batch)
			if res.Error != nil {
				return total, res.Error
			}
			if res.RowsAffected == 0 {
				break
			}
			total += res.RowsAffected
			if res.RowsAffected < int64(batch) {
				break
			}
		}
	}
	return total, nil
}
