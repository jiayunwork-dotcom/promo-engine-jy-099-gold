<script>
  import { onMount } from 'svelte'
  import { navigate } from 'svelte-routing'
  import { api } from '../lib/api.js'

  let rules = []
  let loading = true
  let filterStatus = ''
  let filterType = ''

  const statusLabels = {
    draft: '草稿',
    review: '审核中',
    active: '生效中',
    expired: '已过期'
  }

  const typeLabels = {
    full_reduction: '满减',
    discount: '折扣',
    buy_gift: '买赠',
    nth_item: '第N件优惠',
    cross_store: '跨店满减',
    combo: '组合优惠'
  }

  onMount(loadRules)

  async function loadRules() {
    loading = true
    try {
      const params = {}
      if (filterStatus) params.status = filterStatus
      if (filterType) params.type = filterType
      rules = await api.getRules(params)
    } catch (e) {
      console.error('Failed to load rules:', e)
    } finally {
      loading = false
    }
  }

  function editRule(id) {
    navigate(`/rules/${id}`)
  }

  async function deleteRule(id) {
    if (confirm('确定要删除这条规则吗？')) {
      try {
        await api.deleteRule(id)
        loadRules()
      } catch (e) {
        alert('删除失败')
      }
    }
  }

  async function updateStatus(id, status) {
    try {
      await api.updateRuleStatus(id, status)
      loadRules()
    } catch (e) {
      alert('状态更新失败')
    }
  }
</script>

<div class="page-header">
  <h1>促销规则管理</h1>
</div>

<div class="card">
  <div class="card-header">
    <h2>规则列表</h2>
    <button class="btn btn-primary" on:click={() => navigate('/rules/new')}>
      + 新建规则
    </button>
  </div>

  <div class="form-row" style="margin-bottom: 20px;">
    <div class="form-group" style="margin-bottom: 0;">
      <label>状态筛选</label>
      <select class="form-control" bind:value={filterStatus} on:change={loadRules}>
        <option value="">全部</option>
        <option value="draft">草稿</option>
        <option value="review">审核中</option>
        <option value="active">生效中</option>
        <option value="expired">已过期</option>
      </select>
    </div>
    <div class="form-group" style="margin-bottom: 0;">
      <label>类型筛选</label>
      <select class="form-control" bind:value={filterType} on:change={loadRules}>
        <option value="">全部</option>
        <option value="full_reduction">满减</option>
        <option value="discount">折扣</option>
        <option value="buy_gift">买赠</option>
        <option value="nth_item">第N件优惠</option>
        <option value="cross_store">跨店满减</option>
        <option value="combo">组合优惠</option>
      </select>
    </div>
  </div>

  {#if loading}
    <p>加载中...</p>
  {:else}
    <table>
      <thead>
        <tr>
          <th>规则名称</th>
          <th>类型</th>
          <th>状态</th>
          <th>生效时间</th>
          <th>失效时间</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        {#each rules as rule}
          <tr>
            <td>{rule.name}</td>
            <td><span class="promo-type-badge">{typeLabels[rule.promo_type] || rule.promo_type}</span></td>
            <td><span class="status-badge status-{rule.status}">{statusLabels[rule.status] || rule.status}</span></td>
            <td>{new Date(rule.time_condition.start_time).toLocaleDateString()}</td>
            <td>{new Date(rule.time_condition.end_time).toLocaleDateString()}</td>
            <td>
              <button class="btn btn-sm btn-secondary" on:click={() => editRule(rule.id)}>编辑</button>
              {#if rule.status === 'draft'}
                <button class="btn btn-sm btn-primary" on:click={() => updateStatus(rule.id, 'review')}>提交审核</button>
              {/if}
              {#if rule.status === 'review'}
                <button class="btn btn-sm btn-primary" on:click={() => updateStatus(rule.id, 'active')}>审核通过</button>
              {/if}
              <button class="btn btn-sm btn-danger" on:click={() => deleteRule(rule.id)}>删除</button>
            </td>
          </tr>
        {/each}
        {#if rules.length === 0}
          <tr>
            <td colspan="6" style="text-align: center; color: #999;">暂无规则</td>
          </tr>
        {/if}
      </tbody>
    </table>
  {/if}
</div>
