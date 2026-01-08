<!--
  +page.svelte (business-units/[id])
  Level 2: Team View
  Shows business unit aggregate metrics, technician comparison, and trend chart
-->
<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { 
		MetricsPanel, 
		RangeSelector, 
		LineChart, 
		TechnicianTable,
		MetricSelector 
	} from '$lib/components';
	import { getBusinessUnit } from '$lib/api.js';
	
	// Get the business unit ID from the URL
	$: unitId = $page.params.id;
	
	let businessUnit = null;
	let loading = true;
	let error = null;
	
	let selectedRange = '30d';
	let selectedMetric = 'close_rate';
	
	// Fetch data when component mounts or when range changes
	$: if (unitId) {
		loadData(unitId, selectedRange);
	}
	
	async function loadData(id, range) {
		loading = true;
		error = null;
		
		try {
			businessUnit = await getBusinessUnit(id, range);
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
	
	function handleTechnicianSelect(techId) {
		goto(`/technicians/${techId}`);
	}
</script>

<svelte:head>
	<title>{businessUnit?.name || 'Loading...'} | Performance Dashboard</title>
</svelte:head>

<div class="page">
	<!-- Breadcrumbs -->
	<nav class="breadcrumbs" aria-label="Breadcrumb">
		<a href="/">Business Units</a>
		<span class="separator">/</span>
		<span class="current">{businessUnit?.name || '...'}</span>
	</nav>
	
	{#if loading && !businessUnit}
		<div class="loading-state">
			<div class="skeleton skeleton-header"></div>
			<div class="skeleton skeleton-metrics"></div>
			<div class="skeleton skeleton-chart"></div>
			<div class="skeleton skeleton-table"></div>
		</div>
	{:else if error}
		<div class="error-state card">
			<p class="error-message">Failed to load business unit: {error}</p>
			<button class="btn btn-primary" on:click={() => loadData(unitId, selectedRange)}>
				Retry
			</button>
		</div>
	{:else if businessUnit}
		<!-- Page Header -->
		<header class="page-header">
			<h1 class="page-title">{businessUnit.name}</h1>
			<p class="page-subtitle">Team performance overview and individual metrics</p>
		</header>
		
		<!-- Aggregate Metrics -->
		<section class="section">
			<MetricsPanel metrics={businessUnit.metrics} />
		</section>
		
		<!-- Trend Chart -->
		<section class="section">
			<div class="card">
				<div class="card-header">
					<h2 class="card-title">Team Performance Trend</h2>
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
					data={businessUnit.trend} 
					metricKey={selectedMetric} 
				/>
			</div>
		</section>
		
		<!-- Technician Comparison -->
		<section class="section">
			<div class="section-header">
				<h2 class="section-title">Technicians</h2>
				<p class="section-subtitle">Click a row to view detailed technician stats</p>
			</div>
			<TechnicianTable 
				technicians={businessUnit.technicians} 
				onSelect={handleTechnicianSelect}
			/>
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
		height: 80px;
	}
	
	.skeleton-metrics {
		height: 100px;
	}
	
	.skeleton-chart {
		height: 400px;
	}
	
	.skeleton-table {
		height: 300px;
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
	}
</style>