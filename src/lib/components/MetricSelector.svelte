<!--
  MetricSelector.svelte
  Dropdown for selecting which metric to display in a chart
  
  Props:
  - value: string - Currently selected metric key
  
  Events:
  - change: Dispatched when selection changes
-->
<script>
	import { createEventDispatcher } from 'svelte';
	import { METRIC_DEFINITIONS } from '$lib/api.js';
	
	export let value = 'close_rate';
	
	const dispatch = createEventDispatcher();
	
	const metrics = Object.entries(METRIC_DEFINITIONS).map(([key, def]) => ({
		value: key,
		label: def.label
	}));
	
	function handleChange(event) {
		dispatch('change', event.target.value);
	}
</script>

<div class="metric-selector">
	<label for="metric-select" class="visually-hidden">Select metric</label>
	<select id="metric-select" bind:value on:change={handleChange}>
		{#each metrics as metric}
			<option value={metric.value}>{metric.label}</option>
		{/each}
	</select>
	<span class="select-arrow">▼</span>
</div>

<style>
	.metric-selector {
		position: relative;
		display: inline-block;
	}
	
	select {
		appearance: none;
		padding: var(--space-2) var(--space-8) var(--space-2) var(--space-3);
		font-family: var(--font-sans);
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--color-text-primary);
		background: var(--color-bg-elevated);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		cursor: pointer;
		transition: all var(--transition-fast);
	}
	
	select:hover {
		border-color: var(--color-text-muted);
	}
	
	select:focus {
		outline: none;
		border-color: var(--color-accent);
	}
	
	.select-arrow {
		position: absolute;
		right: var(--space-3);
		top: 50%;
		transform: translateY(-50%);
		font-size: 0.625rem;
		color: var(--color-text-muted);
		pointer-events: none;
	}
	
	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}
</style>