<!--
  TechnicianTable.svelte
  Displays technicians in a sortable table for comparison
  
  Props:
  - technicians: Array<{id, name, metrics}> - List of technicians
  - onSelect: function(id) - Called when a technician row is clicked
-->
<script>
	import { METRIC_DEFINITIONS, formatMetric } from '$lib/api.js';
	
	export let technicians = [];
	export let onSelect = () => {};
	
	let sortKey = 'close_rate';
	let sortDirection = 'desc';
	
	const metricKeys = [
		'close_rate',
		'average_ticket',
		'total_jobs',
		'average_time_on_job',
		'average_estimates_per_job',
		'callback_rate'
	];
	
	$: sortedTechnicians = [...technicians].sort((a, b) => {
		const aVal = a.metrics[sortKey];
		const bVal = b.metrics[sortKey];
		const multiplier = sortDirection === 'desc' ? -1 : 1;
		return (aVal - bVal) * multiplier;
	});
	
	function handleSort(key) {
		if (sortKey === key) {
			sortDirection = sortDirection === 'desc' ? 'asc' : 'desc';
		} else {
			sortKey = key;
			// Default sort direction based on whether higher is better
			sortDirection = METRIC_DEFINITIONS[key].higherIsBetter ? 'desc' : 'asc';
		}
	}
	
	function handleRowClick(tech) {
		onSelect(tech.id);
	}
	
	function handleKeydown(event, tech) {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			handleRowClick(tech);
		}
	}
</script>

<div class="table-wrapper">
	<table>
		<thead>
			<tr>
				<th class="name-col">Technician</th>
				{#each metricKeys as key}
					<th 
						class="metric-col sortable"
						class:sorted={sortKey === key}
						on:click={() => handleSort(key)}
						on:keydown={(e) => e.key === 'Enter' && handleSort(key)}
						tabindex="0"
						role="columnheader"
						aria-sort={sortKey === key ? (sortDirection === 'desc' ? 'descending' : 'ascending') : 'none'}
					>
						<span class="th-content">
							{METRIC_DEFINITIONS[key].label}
							{#if sortKey === key}
								<span class="sort-indicator">
									{sortDirection === 'desc' ? '↓' : '↑'}
								</span>
							{/if}
						</span>
					</th>
				{/each}
			</tr>
		</thead>
		<tbody>
			{#each sortedTechnicians as tech (tech.id)}
				<tr 
					class="clickable"
					on:click={() => handleRowClick(tech)}
					on:keydown={(e) => handleKeydown(e, tech)}
					tabindex="0"
					role="button"
				>
					<td class="name-col">
						<span class="tech-name">{tech.name}</span>
					</td>
					{#each metricKeys as key}
						<td class="metric-col">
							<span class="metric-value" class:mono={true}>
								{formatMetric(key, tech.metrics[key])}
							</span>
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<style>
	.table-wrapper {
		overflow-x: auto;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		background: var(--color-bg-elevated);
	}
	
	table {
		width: 100%;
		border-collapse: collapse;
		min-width: 900px;
	}
	
	th, td {
		padding: var(--space-3) var(--space-4);
		text-align: right;
		border-bottom: 1px solid var(--color-border-subtle);
	}
	
	th {
		font-size: 0.6875rem;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		background: var(--color-bg-card);
		position: sticky;
		top: 0;
	}
	
	th.sortable {
		cursor: pointer;
		user-select: none;
		transition: color var(--transition-fast);
	}
	
	th.sortable:hover {
		color: var(--color-text-secondary);
	}
	
	th.sorted {
		color: var(--color-accent);
	}
	
	.th-content {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
	}
	
	.sort-indicator {
		font-size: 0.75rem;
	}
	
	.name-col {
		text-align: left;
		min-width: 180px;
	}
	
	.metric-col {
		min-width: 100px;
	}
	
	tbody tr {
		transition: background var(--transition-fast);
	}
	
	tbody tr:hover {
		background: var(--color-bg-hover);
	}
	
	tbody tr:focus {
		outline: 2px solid var(--color-accent);
		outline-offset: -2px;
	}
	
	tbody tr:last-child td {
		border-bottom: none;
	}
	
	.tech-name {
		font-weight: 500;
		color: var(--color-text-primary);
	}
	
	.metric-value {
		color: var(--color-text-secondary);
	}
	
	.metric-value.mono {
		font-family: var(--font-mono);
		font-size: 0.875rem;
	}
</style>