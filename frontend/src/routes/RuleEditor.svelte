<script>
  import { onMount } from 'svelte'
  import { navigate } from 'svelte-routing'
  import { api } from '../lib/api.js'

  export let id = 'new'
  $: isNew = id === 'new'

  let currentStep = 0
  const steps = ['选择类型', '配置条件', '设置互斥', '预览效果']

  let rule = {
    name: '',
    description: '',
    promo_type: 'full_reduction',
    config: {},
    scope: { type: 'all' },
    time_condition: {
      start_time: new Date().toISOString().slice(0, 16),
      end_time: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString().slice(0, 16)
    },
    priority: 0,
    created_by: 'admin'
  }

  let conflictResult = null
  let estimationResult = null
  let loading = false

  const promoTypes = [
    { value: 'full_reduction', label: '满减' },
    { value: 'discount', label: '折扣' },
    { value: 'buy_gift', label: '买赠' },
    { value: 'nth_item', label: '第N件优惠' },
    { value: 'cross_store', label: '跨店满减' },
    { value: 'combo', label: '组合优惠' }
  ]

  const scopeTypes = [
    { value: 'all', label: '全场' },
    { value: 'category', label: '指定品类' },
    { value: 'store', label: '指定店铺' },
    { value: 'sku', label: '指定商品' },
    { value: 'user_tag', label: '指定用户标签' }
  ]

  let tierPlaceholder = '{"tiers": [{"threshold": 200, "discount": 30}]}'

  onMount(async () => {
    if (!isNew) {
      try {
        const data = await api.getRule(id)
        rule = data
      } catch (e) {
        alert('加载规则失败')
      }
    }
  })

  function nextStep() {
    if (currentStep < steps.length - 1) {
      currentStep++
      if (currentStep === 3) {
        checkConflicts()
        estimateEffect()
      }
    }
  }

  function prevStep() {
    if (currentStep > 0) {
      currentStep--
    }
  }

  async function checkConflicts() {
    try {
      conflictResult = await api.detectConflicts(rule)
    } catch (e) {
      console.error('Conflict check failed:', e)
    }
  }

  async function estimateEffect() {
    try {
      estimationResult = await api.estimateEffect(rule)
    } catch (e) {
      console.error('Estimation failed:', e)
    }
  }

  async function saveRule() {
    loading = true
    try {
      if (isNew) {
        await api.createRule(rule)
      } else {
        await api.updateRule(id, rule)
      }
      navigate('/rules')
    } catch (e) {
      alert('保存失败')
    } finally {
      loading = false
    }
  }

  function updateConfig(key, value) {
    rule.config = { ...rule.config, [key]: value }
  }
</script>

<div class="page-header">
  <h1>{isNew ? '新建促销规则' : '编辑促销规则'}</h1>
</div>

<div class="card">
  <div class="wizard-steps">
    {#each steps as step, i}
      <div class="wizard-step {i < currentStep ? 'completed' : ''} {i === currentStep ? 'active' : ''}">
        <div class="wizard-step-number">{i + 1}</div>
        <div class="wizard-step-label">{step}</div>
      </div>
    {/each}
  </div>

  {#if currentStep === 0}
    <div class="form-group">
      <label>规则名称</label>
      <input type="text" class="form-control" bind:value={rule.name} placeholder="请输入规则名称">
    </div>
    <div class="form-group">
      <label>规则描述</label>
      <textarea class="form-control" bind:value={rule.description} rows="3" placeholder="请输入规则描述"></textarea>
    </div>
    <div class="form-group">
      <label>促销类型</label>
      <select class="form-control" bind:value={rule.promo_type}>
        {#each promoTypes as type}
          <option value={type.value}>{type.label}</option>
        {/each}
      </select>
    </div>
    <div class="form-group">
      <label>优先级</label>
      <input type="number" class="form-control" bind:value={rule.priority} placeholder="数字越大优先级越高">
    </div>
  {:else if currentStep === 1}
    <div class="form-group">
      <label>适用范围</label>
      <select class="form-control" bind:value={rule.scope.type}>
        {#each scopeTypes as scope}
          <option value={scope.value}>{scope.label}</option>
        {/each}
      </select>
    </div>

    {#if rule.scope.type === 'category'}
      <div class="form-group">
        <label>品类ID（逗号分隔）</label>
        <input type="text" class="form-control" placeholder="例如: 1,2,3">
      </div>
    {:else if rule.scope.type === 'store'}
      <div class="form-group">
        <label>店铺ID（逗号分隔）</label>
        <input type="text" class="form-control" placeholder="例如: 1,2,3">
      </div>
    {:else if rule.scope.type === 'sku'}
      <div class="form-group">
        <label>商品SKU ID（逗号分隔）</label>
        <input type="text" class="form-control" placeholder="例如: 1001,1002,1003">
      </div>
    {/if}

    <div class="form-row">
      <div class="form-group">
        <label>生效时间</label>
        <input type="datetime-local" class="form-control" bind:value={rule.time_condition.start_time}>
      </div>
      <div class="form-group">
        <label>失效时间</label>
        <input type="datetime-local" class="form-control" bind:value={rule.time_condition.end_time}>
      </div>
    </div>

    {#if rule.promo_type === 'full_reduction'}
      <div class="form-group">
        <label>满减阶梯（JSON格式）</label>
        <textarea class="form-control" rows="4" placeholder={tierPlaceholder}></textarea>
      </div>
    {:else if rule.promo_type === 'discount'}
      <div class="form-group">
        <label>折扣率（0-1）</label>
        <input type="number" class="form-control" step="0.01" min="0" max="1" placeholder="例如: 0.8 表示8折">
      </div>
    {:else if rule.promo_type === 'nth_item'}
      <div class="form-row">
        <div class="form-group">
          <label>第N件</label>
          <input type="number" class="form-control" value="2">
        </div>
        <div class="form-group">
          <label>折扣率</label>
          <input type="number" class="form-control" step="0.01" value="0.5">
        </div>
      </div>
    {/if}
  {:else if currentStep === 2}
    <div class="alert alert-info">
      互斥组配置功能可以在互斥组管理页面进行设置。同一互斥组内的规则只能选择一个应用。
    </div>
    <p>当前规则暂未加入任何互斥组</p>
  {:else if currentStep === 3}
    {#if conflictResult}
      <div class="card" style="margin-bottom: 16px;">
        <h3 style="margin-bottom: 12px;">冲突检测</h3>
        {#if conflictResult.warnings && conflictResult.warnings.length > 0}
          <div class="alert alert-warning">
            {#each conflictResult.warnings as warning}
              <p>{warning}</p>
            {/each}
          </div>
        {/if}
        {#if conflictResult.errors && conflictResult.errors.length > 0}
          <div class="alert alert-error">
            {#each conflictResult.errors as error}
              <p>{error}</p>
            {/each}
          </div>
        {/if}
        {#if !conflictResult.hasConflict}
          <p style="color: #27ae60;">✓ 未检测到严重冲突</p>
        {/if}
      </div>
    {/if}

    {#if estimationResult}
      <div class="card">
        <h3 style="margin-bottom: 12px;">效果预估</h3>
        <div class="stats-grid" style="grid-template-columns: repeat(3, 1fr);">
          <div class="stat-card">
            <h3>预估覆盖订单</h3>
            <div class="value">{estimationResult.estimated_orders || 0}</div>
          </div>
          <div class="stat-card">
            <h3>预估优惠总额</h3>
            <div class="value">¥{(estimationResult.estimated_discount || 0).toFixed(2)}</div>
          </div>
          <div class="stat-card">
            <h3>预估GMV变化</h3>
            <div class="value">¥{(estimationResult.estimated_gmv_change || 0).toFixed(2)}</div>
          </div>
        </div>
      </div>
    {/if}
  {/if}

  <div style="display: flex; justify-content: space-between; margin-top: 24px;">
    <div>
      {#if currentStep > 0}
        <button class="btn btn-secondary" on:click={prevStep}>上一步</button>
      {/if}
    </div>
    <div>
      <button class="btn btn-secondary" on:click={() => navigate('/rules')}>取消</button>
      {#if currentStep < steps.length - 1}
        <button class="btn btn-primary" style="margin-left: 8px;" on:click={nextStep}>下一步</button>
      {:else}
        <button class="btn btn-primary" style="margin-left: 8px;" on:click={saveRule} disabled={loading}>
          {loading ? '保存中...' : '保存规则'}
        </button>
      {/if}
    </div>
  </div>
</div>
