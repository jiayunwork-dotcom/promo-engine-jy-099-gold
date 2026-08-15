<script>
  import { api } from '../lib/api.js'

  let items = [
    { sku_id: 1, sku_name: '商品A', store_id: 1, price: 100, quantity: 2 }
  ]
  let couponCode = ''
  let result = null
  let loading = false

  function addItem() {
    items.push({
      sku_id: items.length + 1,
      sku_name: `商品${String.fromCharCode(65 + items.length)}`,
      store_id: 1,
      price: 100,
      quantity: 1
    })
    items = items
  }

  function removeItem(index) {
    items.splice(index, 1)
    items = items
  }

  async function calculate() {
    loading = true
    result = null
    try {
      const cart = {
        user_id: 'test_user',
        items: items.map(i => ({
          sku_id: i.sku_id,
          sku_name: i.sku_name,
          store_id: i.store_id,
          price: i.price,
          quantity: i.quantity
        })),
        coupon_code: couponCode.toUpperCase()
      }
      result = await api.calculatePrice(cart)
    } catch (e) {
      alert('计算失败: ' + e.message)
    } finally {
      loading = false
    }
  }

  function formatCurrency(value) {
    return '¥' + (value || 0).toFixed(2)
  }
</script>

<div class="page-header">
  <h1>价格模拟器</h1>
  <p>输入商品列表，模拟计算优惠结果</p>
</div>

<div class="simulator-container">
  <div class="card">
    <div class="card-header">
      <h2>购物车商品</h2>
      <button class="btn btn-primary btn-sm" on:click={addItem}>+ 添加商品</button>
    </div>
    
    {#each items as item, index}
      <div style="display: grid; grid-template-columns: 2fr 1fr 1fr auto; gap: 12px; margin-bottom: 12px; align-items: end;">
        <div class="form-group" style="margin-bottom: 0;">
          <label>商品名称</label>
          <input type="text" class="form-control" bind:value={item.sku_name}>
        </div>
        <div class="form-group" style="margin-bottom: 0;">
          <label>单价</label>
          <input type="number" class="form-control" bind:value={item.price} min="0" step="0.01">
        </div>
        <div class="form-group" style="margin-bottom: 0;">
          <label>数量</label>
          <input type="number" class="form-control" bind:value={item.quantity} min="1">
        </div>
        <button class="btn btn-danger btn-sm" on:click={() => removeItem(index)}>删除</button>
      </div>
    {/each}

    <div class="form-group" style="margin-top: 16px;">
      <label>优惠券码（可选）</label>
      <input type="text" class="form-control" bind:value={couponCode} placeholder="输入优惠券码，如 ABC123XYZ789">
    </div>

    <button class="btn btn-primary" on:click={calculate} disabled={loading || items.length === 0} style="width: 100%; margin-top: 16px;">
      {loading ? '计算中...' : '计算最优优惠'}
    </button>
  </div>

  <div class="card">
    <div class="card-header">
      <h2>计算结果</h2>
    </div>

    {#if result}
      <div>
        {#each result.items as item}
          <div class="result-item">
            <div>
              <strong>{item.sku_name}</strong>
              <span style="color: #888; margin-left: 8px;">x{item.quantity}</span>
            </div>
            <div>
              <span style="text-decoration: line-through; color: #999; margin-right: 8px;">
                {formatCurrency(item.original_price)}
              </span>
              <span style="color: #e74c3c;">{formatCurrency(item.pay_price)}</span>
            </div>
          </div>
          {#if item.discount_amount > 0}
            <div style="color: #27ae60; font-size: 12px; margin-bottom: 8px;">
              优惠: -{formatCurrency(item.discount_amount)}
              {#if item.promo_rule_ids && item.promo_rule_ids.length > 0}
                <span class="tag">规则ID: {item.promo_rule_ids.join(', ')}</span>
              {/if}
            </div>
          {/if}
        {/each}

        {#if result.gift_items && result.gift_items.length > 0}
          <div style="border-top: 1px solid #eee; padding-top: 16px; margin-top: 8px;">
            <h4 style="margin-bottom: 12px; color: #27ae60;">🎁 赠品</h4>
            {#each result.gift_items as gift}
              <div class="result-item">
                <span>{gift.sku_name} x{gift.quantity}</span>
                <span style="color: #27ae60;">免费</span>
              </div>
            {/each}
          </div>
        {/if}

        <div style="border-top: 2px solid #eee; margin-top: 16px;">
          <div class="result-total">
            <span>商品原价</span>
            <span>{formatCurrency(result.total_original)}</span>
          </div>
          {#if result.coupon_discount > 0}
            <div class="result-total" style="color: #9b59b6; padding-top: 0;">
              <span>🎟️ 优惠券优惠 ({result.coupon_code})</span>
              <span>-{formatCurrency(result.coupon_discount)}</span>
            </div>
          {/if}
          <div class="result-total" style="color: #e74c3c; padding-top: 0;">
            <span>总优惠</span>
            <span>-{formatCurrency(result.total_discount)}</span>
          </div>
          <div class="result-total" style="font-size: 22px; padding-top: 8px;">
            <span>实付金额</span>
            <span style="color: #e74c3c;">{formatCurrency(result.total_pay)}</span>
          </div>
        </div>
      </div>
    {:else}
      <p style="color: #999; text-align: center; padding: 40px 0;">
        添加商品后点击"计算最优优惠"查看结果
      </p>
    {/if}
  </div>
</div>
