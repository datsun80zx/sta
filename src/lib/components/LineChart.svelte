<!--
  LineChart.svelte
  Renders a line chart using Chart.js
  
  Props:
  - data: Array<{date: string, metrics: object}> - Trend data points
  - metricKey: string - Which metric to display (e.g., 'close_rate')
-->
<script>
	import { onMount, onDestroy, afterUpdate } from 'svelte';
	import { METRIC_DEFINITIONS } from '$lib/api.js';
	
	export let data = [];
	export let metricKey = 'close_rate';
	
	let canvas;
	let chart = null;
	
	$: metricDef = METRIC_DEFINITIONS[metricKey];
	
	// Reactive update when data or metric changes
	$: if (chart && data) {
		updateChart(data, metricKey);
	}
	
	onMount(async () => {
		// Dynamic import to avoid SSR issues
		const { Chart, registerables } = await import('chart.js');
		Chart.register(...registerables);
		
		const ctx = canvas.getContext('2d');
		
		chart = new Chart(ctx, {
			type: 'line',
			data: {
				labels: data.map(d => formatDate(d.date)),
				datasets: [{
					label: metricDef?.label || metricKey,
					data: data.map(d => d.metrics[metricKey]),
					borderColor: getComputedStyle(document.documentElement)
						.getPropertyValue('--chart-line').trim(),
					backgroundColor: 'transparent',
					borderWidth: 2,
					tension: 0.3,
					pointRadius: 0,
					pointHoverRadius: 6,
					pointHoverBackgroundColor: getComputedStyle(document.documentElement)
						.getPropertyValue('--chart-line').trim(),
					pointHoverBorderColor: '#fff',
					pointHoverBorderWidth: 2
				}]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				interaction: {
					mode: 'index',
					intersect: false
				},
				plugins: {
					legend: {
						display: false
					},
					tooltip: {
						backgroundColor: getComputedStyle(document.documentElement)
							.getPropertyValue('--color-bg-card').trim(),
						titleColor: getComputedStyle(document.documentElement)
							.getPropertyValue('--color-text-primary').trim(),
						bodyColor: getComputedStyle(document.documentElement)
							.getPropertyValue('--color-text-secondary').trim(),
						borderColor: getComputedStyle(document.documentElement)
							.getPropertyValue('--color-border').trim(),
						borderWidth: 1,
						padding: 12,
						displayColors: false,
						callbacks: {
							label: (context) => {
								const value = context.parsed.y;
								return metricDef ? metricDef.format(value) : value;
							}
						}
					}
				},
				scales: {
					x: {
						grid: {
							color: getComputedStyle(document.documentElement)
								.getPropertyValue('--chart-grid').trim(),
							drawBorder: false
						},
						ticks: {
							color: getComputedStyle(document.documentElement)
								.getPropertyValue('--color-text-muted').trim(),
							font: {
								family: getComputedStyle(document.documentElement)
									.getPropertyValue('--font-sans').trim(),
								size: 11
							},
							maxTicksLimit: 8
						}
					},
					y: {
						grid: {
							color: getComputedStyle(document.documentElement)
								.getPropertyValue('--chart-grid').trim(),
							drawBorder: false
						},
						ticks: {
							color: getComputedStyle(document.documentElement)
								.getPropertyValue('--color-text-muted').trim(),
							font: {
								family: getComputedStyle(document.documentElement)
									.getPropertyValue('--font-mono').trim(),
								size: 11
							},
							callback: (value) => {
								if (metricKey.includes('rate')) {
									return `${(value * 100).toFixed(0)}%`;
								}
								if (metricKey === 'average_ticket') {
									return `$${value}`;
								}
								return value;
							}
						},
						beginAtZero: false
					}
				}
			}
		});
	});
	
	onDestroy(() => {
		if (chart) {
			chart.destroy();
		}
	});
	
	function updateChart(newData, key) {
		if (!chart) return;
		
		chart.data.labels = newData.map(d => formatDate(d.date));
		chart.data.datasets[0].data = newData.map(d => d.metrics[key]);
		chart.data.datasets[0].label = METRIC_DEFINITIONS[key]?.label || key;
		chart.update('none');
	}
	
	function formatDate(dateStr) {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}
</script>

<div class="chart-container">
	<canvas bind:this={canvas}></canvas>
</div>

<style>
	.chart-container {
		position: relative;
		width: 100%;
		height: 300px;
	}
</style>