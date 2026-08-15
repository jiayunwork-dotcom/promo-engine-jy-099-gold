<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'

  let groups = []
  let rules = []
  let loading = true
  let showCreateModal = false
  let newGroup = { name: '', description: '' }

  onMount(loadData)

  async function loadData() {
    loading = true
    try {
      const [groupsData, rulesData] = await Promise.all([
        api.getMutexGroups(),
        api.getRules()
      ])
      groups = groupsData
      rules = rulesData
    } catch (e) {
      console.error('Failed to load data:', e)
    } finally {
      loading = false
    }
  }

  async function createGroup() {
    if (!newGroup.name) return
    
    try {
      await api.createMutexGroup(newGroup)
      newGroup = { name: '', description: '' }
      showCreateModal = false
      loadData()
    } catch (e) {
      alert('创建失败')
    }
  }

  async function deleteGroup(id) {
    if (confirm('确定要删除这个互斥组吗？')) {
      try {
        await api.deleteMutexGroup(id)
        loadData()
      } catch (e) {
        alert('删除失败')
      }
    }
  }

  async function addRuleToGroup(groupId) {
    const ruleId = prompt('请输入要添加的规则ID:')
    if (ruleId) {
      try {
        await api.addRuleToMutexGroup(groupId, parseInt(ruleId))
        loadData()
      } catch (e) {
        alert('添加失败')
      }
    }
  }

  async function removeRuleFromGroup(groupId, ruleId) {
    if (confirm('确定要移除这条规则吗？')) {
      try {
        await api.removeRuleFromMutexGroup(groupId, ruleId)
        loadData()
      } catch (e) {
        alert('移除失败')
      }
    }
  }
</script>

<div class="page-header">
  <h1>互斥组管理</h1>
  <p>配置促销规则的互斥关系</p>
</div>

<div class="card">
  <div class="card-header">
    <h2>互斥组列表</h2>
    <button class="btn btn-primary" on:click={() => showCreateModal = true}>
      + 新建互斥组
    </button>
  </div>

  {#if loading}
    <p>加载中...</p>
  {:else}
    <table>
      <thead>
        <tr>
          <th>互斥组名称</th>
          <th>描述</th>
          <th>规则数量</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        {#each groups as group}
          <tr>
            <td>{group.name}</td>
            <td>{group.description || '-'}</td>
            <td>
              {#if Array.isArray(group.rules)}
                {group.rules.length}
              {:else}
                0
              {/if}
            </td>
            <td>
              <button class="btn btn-sm btn-secondary" on:click={() => addRuleToGroup(group.id)}>
                添加规则
              </button>
              <button class="btn btn-sm btn-danger" on:click={() => deleteGroup(group.id)}>
                删除
              </button>
            </td>
          </tr>
          {#if Array.isArray(group.rules) && group.rules.length > 0}
            <tr>
              <td colspan="4" style="background: #f8f9fa; padding: 12px 24px;">
                <strong>组内规则：</strong>
                {#each group.rules as rule}
                  <span class="tag" style="margin: 4px;">
                    {rule.name}
                    <button 
                      style="margin-left: 6px; background: none; border: none; cursor: pointer; color: #e74c3c;"
                      on:click={() => removeRuleFromGroup(group.id, rule.id)}
                    >×</button>
                  </span>
                {/each}
              </td>
            </tr>
          {/if}
        {/each}
        {#if groups.length === 0}
          <tr>
            <td colspan="4" style="text-align: center; color: #999;">暂无互斥组</td>
          </tr>
        {/if}
      </tbody>
    </table>
  {/if}
</div>

{#if showCreateModal}
  <div style="position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000;">
    <div class="card" style="width: 400px; margin: 0;">
      <div class="card-header">
        <h2>新建互斥组</h2>
      </div>
      <div class="form-group">
        <label>组名称</label>
        <input type="text" class="form-control" bind:value={newGroup.name} placeholder="例如: 满减类促销互斥">
      </div>
      <div class="form-group">
        <label>描述</label>
        <textarea class="form-control" bind:value={newGroup.description} rows="3" placeholder="描述该互斥组的作用"></textarea>
      </div>
      <div style="display: flex; justify-content: flex-end; gap: 8px;">
        <button class="btn btn-secondary" on:click={() => showCreateModal = false}>取消</button>
        <button class="btn btn-primary" on:click={createGroup}>创建</button>
      </div>
    </div>
  </div>
{/if}
