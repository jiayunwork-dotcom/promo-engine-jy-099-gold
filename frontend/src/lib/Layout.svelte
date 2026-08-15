<script>
  import { link } from 'svelte-routing'

  export let currentPath = '/'

  const navItems = [
    { path: '/', label: '仪表盘', icon: '📊' },
    { path: '/rules', label: '促销规则', icon: '📋' },
    { path: '/calendar', label: '日历视图', icon: '📅' },
    { path: '/simulator', label: '价格模拟器', icon: '🧮' },
    { path: '/mutex-groups', label: '互斥组', icon: '🔗' },
    { path: '/coupons', label: '优惠券管理', icon: '🎟️' },
  ]

  function isActive(itemPath) {
    if (itemPath === '/') {
      return currentPath === '/'
    }
    return currentPath === itemPath || currentPath.startsWith(itemPath + '/')
  }
</script>

<div class="app-container">
  <aside class="sidebar">
    <div class="sidebar-logo">
      🛒 促销引擎
    </div>
    <nav class="sidebar-nav">
      {#each navItems as item}
        <a use:link href={item.path} class:active={isActive(item.path)}>
          {item.icon} {item.label}
        </a>
      {/each}
    </nav>
  </aside>
  <main class="main-content">
    <slot />
  </main>
</div>
