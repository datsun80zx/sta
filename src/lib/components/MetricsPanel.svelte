<!--
  MetricsPanel.svelte
  Displays all 6 metrics in a responsive grid layout
  
  Props:
  - metrics: object - Contains all 6 metric values
  - trends: object (optional) - Contains trend percentages for each metric
-->
<script>
	import MetricCard from './MetricCard.svelte';
	import { METRIC_DEFINITIONS, formatMetric } from '$lib/api.js';
	
	export let metrics;
	export let trends = null;
	
	// Order the metrics for display
	const metricKeys = [
		'close_rate',
		'average_ticket', 
		'total_jobs',
		'average_time_on_job',
		'average_estimates_per_job',
		'callback_rate'
	];
</script>

<div class="metrics-panel">
	{#each metricKeys as key}
		{@const def = METRIC_DEFINITIONS[key]}
		<MetricCard
			label={def.label}
			value={formatMetric(key, metrics[key])}
			trend={trends?.[key] ?? null}
			inverted={!def.higherIsBetter}
		/>
	{/each}
</div>

<style>
	.metrics-panel {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: var(--space-4);
	}
	
	/* Responsive breakpoints */
	@media (max-width: 1200px) {
		.metrics-panel {
			grid-template-columns: repeat(3, 1fr);
		}
	}
	
	@media (max-width: 768px) {
		.metrics-panel {
			grid-template-columns: repeat(2, 1fr);
		}
	}
	
	@media (max-width: 480px) {
		.metrics-panel {
			grid-template-columns: 1fr;
		}
	}
</style>