<!--
  RangeSelector.svelte
  Button group for selecting time ranges
  
  Props:
  - value: string - Currently selected range (7d, 30d, 90d, ytd)
  
  Events:
  - change: Dispatched when selection changes, detail contains new value
-->
<script>
	import { createEventDispatcher } from 'svelte';
	import { TIME_RANGES } from '$lib/api.js';
	
	export let value = '30d';
	
	const dispatch = createEventDispatcher();
	
	function select(range) {
		if (range !== value) {
			value = range;
			dispatch('change', range);
		}
	}
</script>

<div class="range-selector" role="group" aria-label="Time range selection">
	{#each TIME_RANGES as range}
		<button
			class="range-btn"
			class:active={value === range.value}
			on:click={() => select(range.value)}
			aria-pressed={value === range.value}
		>
			{range.label}
		</button>
	{/each}
</div>

<style>
	.range-selector {
		display: inline-flex;
		background: var(--color-bg-elevated);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-1);
		gap: var(--space-1);
	}
	
	.range-btn {
		padding: var(--space-2) var(--space-3);
		font-family: var(--font-sans);
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--color-text-secondary);
		background: transparent;
		border: none;
		border-radius: var(--radius-sm);
		cursor: pointer;
		transition: all var(--transition-fast);
	}
	
	.range-btn:hover:not(.active) {
		color: var(--color-text-primary);
		background: var(--color-bg-hover);
	}
	
	.range-btn.active {
		color: white;
		background: var(--color-accent);
	}
</style>