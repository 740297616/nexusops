<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createServerGroup, createServerTag, deleteServerGroup, deleteServerTag, listServerGroups, listServerTags } from '../api/system'
import type { ServerGroup, ServerTag } from '../types/api'
const groups=ref<ServerGroup[]>([]);const tags=ref<ServerTag[]>([])
async function load(){try{groups.value=(await listServerGroups()).items;tags.value=await listServerTags()}catch(e){ElMessage.error(e instanceof Error?e.message:'加载失败')}}
async function addGroup(){try{const code=await ElMessageBox.prompt('分组标识','新增分组');const name=await ElMessageBox.prompt('分组名称','新增分组');await createServerGroup({code:code.value,name:name.value,description:''});await load()}catch(e){if(e!=='cancel')ElMessage.error(e instanceof Error?e.message:'创建失败')}}
async function addTag(){try{const code=await ElMessageBox.prompt('标签标识','新增标签');const name=await ElMessageBox.prompt('标签名称','新增标签');await createServerTag({code:code.value,name:name.value});await load()}catch(e){if(e!=='cancel')ElMessage.error(e instanceof Error?e.message:'创建失败')}}
async function removeGroup(g:ServerGroup){try{await ElMessageBox.confirm(`确认删除 ${g.name}？`,'确认操作');await deleteServerGroup(g.id);await load()}catch(e){if(e!=='cancel')ElMessage.error(e instanceof Error?e.message:'删除失败')}}
async function removeTag(t:ServerTag){try{await ElMessageBox.confirm(`确认删除 ${t.name}？`,'确认操作');await deleteServerTag(t.id);await load()}catch(e){if(e!=='cancel')ElMessage.error(e instanceof Error?e.message:'删除失败')}}
onMounted(load)
</script>
<template><div class="page-heading"><div><div class="eyebrow">INFRASTRUCTURE / GROUPS & TAGS</div><h1>分组与标签</h1><p>维护服务器筛选维度</p></div></div><section class="section-block"><div class="page-heading"><h2>分组</h2><el-button @click="addGroup">新增分组</el-button></div><el-table :data="groups"><el-table-column prop="code" label="标识"/><el-table-column prop="name" label="名称"/><el-table-column label="操作"><template #default="{row}"><el-button link type="danger" @click="removeGroup(row)">删除</el-button></template></el-table-column></el-table></section><section class="section-block"><div class="page-heading"><h2>标签</h2><el-button @click="addTag">新增标签</el-button></div><el-table :data="tags"><el-table-column prop="code" label="标识"/><el-table-column prop="name" label="名称"/><el-table-column label="操作"><template #default="{row}"><el-button link type="danger" @click="removeTag(row)">删除</el-button></template></el-table-column></el-table></section></template>
