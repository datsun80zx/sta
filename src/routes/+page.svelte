<!--
  +page.svelte (root)
  Level 1: Business Units Overview
  Shows all business units with their aggregate metrics
-->
<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { MetricsPanel } from '$lib/components';
	import { getBusinessUnits, METRIC_DEFINITIONS, formatMetric } from '$lib/api.js';
	
	let businessUnits = [];
	let loading = true;
	let error = null;
	
	onMount(async () => {
		try {
			const data = await getBusinessUnits();
			businessUnits = data.business_units;
		} catch (err) {
			error = err.message;
		} finally {
			loading = false;
		}
	});
	
	function navigateToUnit(id) {
		goto(`/business-units/${id}`);
	}
	
	const metricKeys = [
		'close_rate',
		'average_ticket',
		'total_jobs',
		'average_time_on_job',
		'average_estimates_per_job',
		'callback_rate'
	];
</script>

<svelte:head>
	<title>Performance Dashboard</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<h1 class="page-title">Business Units</h1>
		<p class="page-subtitle">Overview of performance metrics across all departments</p>
	</header>
	
	{#if loading}
		<div class="loading-grid">
			{#each Array(4) as _}
				<div class="card skeleton-card">
					<div class="skeleton skeleton-title"></div>
					<div class="skeleton-metrics">
						{#each Array(6) as _}
							<div class="skeleton skeleton-metric"></div>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	{:else if error}
		<div class="error-state card">
			<p class="error-message">Failed to load business units: {error}</p>
			<button class="btn btn-primary" on:click={() => location.reload()}>
				Retry
			</button>
		</div>
	{:else}
		<div class="units-grid">
			{#each businessUnits as unit (unit.id)}
				<button 
					class="unit-card card"
					on:click={() => navigateToUnit(unit.id)}
				>
					<h2 class="unit-name">{unit.name}</h2>
					<div class="unit-metrics">
						{#each metricKeys as key}
							<div class="unit-metric">
								<span class="metric-label">{METRIC_DEFINITIONS[key].label}</span>
								<span class="metric-value">{formatMetric(key, unit.metrics[key])}</span>
							</div>
						{/each}
					</div>
					<span class="unit-arrow">→</span>
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.page {
		max-width: 1200px;
	}
	
	.loading-grid,
	.units-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: var(--space-6);
	}
	
	.unit-card {
		position: relative;
		text-align: left;
		cursor: pointer;
		border: 1px solid var(--color-border);
		transition: all var(--transition-base);
	}
	
	.unit-card:hover {
		border-color: var(--color-accent);
		transform: translateY(-2px);
	}
	
	.unit-card:focus {
		outline: none;
		border-color: var(--color-accent);
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.2);
	}
	
	.unit-name {
		font-size: 1.25rem;
		font-weight: 600;
		color: var(--color-text-primary);
		margin-bottom: var(--space-5);
	}
	
	.unit-metrics {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: var(--space-4);
	}
	
	.unit-metric {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}
	
	.metric-label {
		font-size: 0.6875rem;
		font-weight: 500;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	
	.metric-value {
		font-family: var(--font-mono);
		font-size: 1rem;
		font-weight: 500;
		color: var(--color-text-primary);
	}
	
	.unit-arrow {
		position: absolute;
		top: var(--space-4);
		right: var(--space-4);
		font-size: 1.25rem;
		color: var(--color-text-muted);
		transition: all var(--transition-fast);
	}
	
	.unit-card:hover .unit-arrow {
		color: var(--color-accent);
		transform: translateX(4px);
	}
	
	/* Skeleton styles */
	.skeleton-card {
		height: 240px;
	}
	
	.skeleton-title {
		height: 24px;
		width: 60%;
		margin-bottom: var(--space-5);
	}
	
	.skeleton-metrics {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: var(--space-4);
	}
	
	.skeleton-metric {
		height: 48px;
	}
	
	/* Error state */
	.error-state {
		text-align: center;
		padding: var(--space-10);
	}
	
	.error-message {
		color: var(--color-danger);
		margin-bottom: var(--space-4);
	}
</style>