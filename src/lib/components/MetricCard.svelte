<!--
  MetricCard.svelte
  Displays a single metric with its label and formatted value
  
  Props:
  - label: string - Display name for the metric
  - value: string - Pre-formatted value to display
  - trend: number (optional) - Percentage change, positive or negative
  - inverted: boolean (optional) - If true, negative trends are good (like callback rate)
-->
<script>
	export let label;
	export let value;
	export let trend = null;
	export let inverted = false;
	
	$: trendDirection = trend > 0 ? 'up' : trend < 0 ? 'down' : 'neutral';
	$: trendIsGood = inverted ? trend < 0 : trend > 0;
</script>

<div class="metric-card">
	<span class="metric-label">{label}</span>
	<span class="metric-value">{value}</span>
	{#if trend !== null}
		<span 
			class="metric-trend" 
			class:positive={trendIsGood && trend !== 0}
			class:negative={!trendIsGood && trend !== 0}
		>
			{#if trendDirection === 'up'}↑{:else if trendDirection === 'down'}↓{/if}
			{Math.abs(trend).toFixed(1)}%
		</span>
	{/if}
</div>

<style>
	.metric-card {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		padding: var(--space-4) var(--space-5);
		background: var(--color-bg-elevated);
		border: 1px solid var(--color-border-subtle);
		border-radius: var(--radius-md);
	}
	
	.metric-label {
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	
	.metric-value {
		font-family: var(--font-mono);
		font-size: 1.5rem;
		font-weight: 600;
		color: var(--color-text-primary);
	}
	
	.metric-trend {
		font-size: 0.75rem;
		font-weight: 500;
	}
	
	.metric-trend.positive {
		color: var(--color-success);
	}
	
	.metric-trend.negative {
		color: var(--color-danger);
	}
</style>