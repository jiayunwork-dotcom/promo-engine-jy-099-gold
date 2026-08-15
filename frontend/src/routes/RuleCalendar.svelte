<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'

  let rules = []
  let loading = true
  let currentDate = new Date()
  let calendarDays = []

  const typeLabels = {
    full_reduction: '满减',
    discount: '折扣',
    buy_gift: '买赠',
    nth_item: '第N件',
    cross_store: '跨店',
    combo: '组合'
  }

  onMount(async () => {
    try {
      rules = await api.getRules()
      generateCalendar()
    } catch (e) {
      console.error('Failed to load rules:', e)
    } finally {
      loading = false
    }
  })

  function generateCalendar() {
    const year = currentDate.getFullYear()
    const month = currentDate.getMonth()
    const firstDay = new Date(year, month, 1)
    const lastDay = new Date(year, month + 1, 0)
    const startDay = firstDay.getDay()
    const daysInMonth = lastDay.getDate()

    calendarDays = []
    
    for (let i = 0; i < startDay; i++) {
      calendarDays.push({ day: null, events: [] })
    }

    for (let day = 1; day <= daysInMonth; day++) {
      const date = new Date(year, month, day)
      const events = rules.filter(rule => {
        const start = new Date(rule.time_condition.start_time)
        const end = new Date(rule.time_condition.end_time)
        return date >= start && date <= end
      })
      calendarDays.push({ day, date, events })
    }
  }

  function prevMonth() {
    currentDate = new Date(currentDate.getFullYear(), currentDate.getMonth() - 1, 1)
    generateCalendar()
  }

  function nextMonth() {
    currentDate = new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 1)
    generateCalendar()
  }
</script>

<div class="page-header">
  <h1>日历视图</h1>
  <p>查看各促销规则的生效时间</p>
</div>

<div class="card">
  <div class="card-header">
    <button class="btn btn-secondary btn-sm" on:click={prevMonth}>← 上月</button>
    <h2>{currentDate.getFullYear()}年{currentDate.getMonth() + 1}月</h2>
    <button class="btn btn-secondary btn-sm" on:click={nextMonth}>下月 →</button>
  </div>

  {#if loading}
    <p>加载中...</p>
  {:else}
    <div class="calendar-grid" style="grid-template-columns: repeat(7, 1fr); margin-bottom: 8px;">
      <div class="calendar-day-header">日</div>
      <div class="calendar-day-header">一</div>
      <div class="calendar-day-header">二</div>
      <div class="calendar-day-header">三</div>
      <div class="calendar-day-header">四</div>
      <div class="calendar-day-header">五</div>
      <div class="calendar-day-header">六</div>
    </div>
    <div class="calendar-grid">
      {#each calendarDays as dayData}
        <div class="calendar-day">
          {#if dayData.day}
            <div class="calendar-day-header">{dayData.day}</div>
            {#each dayData.events as event}
              <div class="calendar-event {event.promo_type}" title="{event.name}">
                {typeLabels[event.promo_type] || event.promo_type}
              </div>
            {/each}
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
