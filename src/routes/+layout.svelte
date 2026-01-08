<!--
  +layout.svelte
  Main application layout - wraps all pages with header and navigation
-->
<script>
	import '../app.css';
</script>

<div class="app">
	<header class="app-header">
		<div class="container header-content">
			<a href="/" class="logo">
				<span class="logo-icon">◆</span>
				<span class="logo-text">Performance Dashboard</span>
			</a>
		</div>
	</header>
	
	<main class="app-main">
		<div class="container">
			<slot />
		</div>
	</main>
</div>

<style>
	.app {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}
	
	.app-header {
		position: sticky;
		top: 0;
		z-index: 100;
		background: var(--color-bg);
		border-bottom: 1px solid var(--color-border);
		backdrop-filter: blur(8px);
	}
	
	.header-content {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 64px;
	}
	
	.logo {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		color: var(--color-text-primary);
		font-weight: 600;
		font-size: 1rem;
	}
	
	.logo:hover {
		color: var(--color-text-primary);
	}
	
	.logo-icon {
		color: var(--color-accent);
		font-size: 1.25rem;
	}
	
	.app-main {
		flex: 1;
		padding: var(--space-8) 0 var(--space-12);
	}
</style>