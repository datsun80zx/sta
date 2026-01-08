<!--
  +page.svelte (technicians/[id])
  Level 3: Technician View
  Shows individual technician lifetime metrics and trend chart
-->
<script>
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { 
		MetricsPanel, 
		RangeSelector, 
		LineChart,
		MetricSelector 
	} from '$lib/components';
	import { getTechnician } from '$lib/api.js';
	
	// Get the technician ID from the URL
	$: techId = $page.params.id;
	
	let technician = null;
	let loading = true;
	let error = null;
	
	let selectedRange = '30d';
	let selectedMetric = 'close_rate';
	
	// Fetch data when component mounts or when range changes
	$: if (techId) {
		loadData(techId, selectedRange);
	}
	
	async function loadData(id, range) {
		loading = true;
		error = null;
		
		try {
			technician = await getTechnician(id, range);
		} catch (err) {
			error = err.message;
		} finally {
			loading = false;
		}
	}
	
	function handleRangeChange(event) {
		selectedRange = event.detail;
	}
	
	function handleMetricChange(event) {
		selectedMetric = event.detail;
	}
</script>

<svelte:head>
	<title>{technician?.name || 'Loading...'} | Performance Dashboard</title>
</svelte:head>

<div class="page">
	<!-- Breadcrumbs -->
	<nav class="breadcrumbs" aria-label="Breadcrumb">
		<a href="/">Business Units</a>
		<span class="separator">/</span>
		{#if technician?.business_unit}
			<a href="/business-units/{technician.business_unit.id}">
				{technician.business_unit.name}
			</a>
			<span class="separator">/</span>
		{/if}
		<span class="current">{technician?.name || '...'}</span>
	</nav>
	
	{#if loading && !technician}
		<div class="loading-state">
			<div class="skeleton skeleton-header"></div>
			<div class="skeleton skeleton-metrics"></div>
			<div class="skeleton skeleton-chart"></div>
		</div>
	{:else if error}
		<div class="error-state card">
			<p class="error-message">Failed to load technician: {error}</p>
			<button class="btn btn-primary" on:click={() => loadData(techId, selectedRange)}>
				Retry
			</button>
		</div>
	{:else if technician}
		<!-- Page Header -->
		<header class="page-header">
			<div class="tech-header">
				<div class="tech-avatar">
					{technician.name.split(' ').map(n => n[0]).join('')}
				</div>
				<div class="tech-info">
					<h1 class="page-title">{technician.name}</h1>
					<p class="page-subtitle">{technician.business_unit.name}</p>
				</div>
			</div>
		</header>
		
		<!-- Lifetime Metrics -->
		<section class="section">
			<div class="section-header">
				<h2 class="section-title">Lifetime Performance</h2>
				<p class="section-subtitle">Aggregate metrics across all time</p>
			</div>
			<MetricsPanel metrics={technician.lifetime_metrics} />
		</section>
		
		<!-- Trend Chart -->
		<section class="section">
			<div class="card">
				<div class="card-header">
					<h2 class="card-title">Performance Trend</h2>
					<div class="chart-controls">
						<MetricSelector 
							value={selectedMetric} 
							on:change={handleMetricChange} 
						/>
						<RangeSelector 
							value={selectedRange} 
							on:change={handleRangeChange} 
						/>
					</div>
				</div>
				<LineChart 
					data={technician.trend} 
					metricKey={selectedMetric} 
				/>
			</div>
		</section>
	{/if}
</div>

<style>
	.page {
		max-width: 1200px;
	}
	
	.section {
		margin-bottom: var(--space-8);
	}
	
	.section-header {
		margin-bottom: var(--space-4);
	}
	
	.section-title {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--color-text-primary);
		margin-bottom: var(--space-1);
	}
	
	.section-subtitle {
		font-size: 0.875rem;
		color: var(--color-text-muted);
	}
	
	.tech-header {
		display: flex;
		align-items: center;
		gap: var(--space-5);
		margin-bottom: var(--space-2);
	}
	
	.tech-avatar {
		width: 64px;
		height: 64px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--color-accent);
		color: white;
		font-size: 1.25rem;
		font-weight: 600;
		border-radius: 50%;
	}
	
	.tech-info {
		flex: 1;
	}
	
	.tech-info .page-title {
		margin-bottom: var(--space-1);
	}
	
	.chart-controls {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}
	
	/* Loading skeleton styles */
	.loading-state {
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
	}
	
	.skeleton-header {
		height: 100px;
	}
	
	.skeleton-metrics {
		height: 100px;
	}
	
	.skeleton-chart {
		height: 400px;
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
	
	/* Responsive adjustments */
	@media (max-width: 768px) {
		.chart-controls {
			flex-direction: column;
			align-items: stretch;
		}
		
		.card-header {
			flex-direction: column;
			align-items: flex-start;
			gap: var(--space-4);
		}
		
		.tech-header {
			flex-direction: column;
			text-align: center;
		}
	}
</style>