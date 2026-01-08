/**
 * API client for the dashboard backend
 * 
 * Base URL is proxied through Vite in development (see vite.config.js)
 * In production, adjust the base URL as needed
 * 
 * Set USE_MOCK_DATA to true to use mock data during frontend development
 * before the Go backend is ready.
 */

const BASE_URL = '/api';

// Toggle this to switch between mock and real API
const USE_MOCK_DATA = true;

// Import mock functions for development
import { mockGetBusinessUnits, mockGetBusinessUnit, mockGetTechnician } from './mock-data.js';

/**
 * Generic fetch wrapper with error handling
 */
async function fetchAPI(endpoint, options = {}) {
	const url = `${BASE_URL}${endpoint}`;
	
	try {
		const response = await fetch(url, {
			headers: {
				'Content-Type': 'application/json',
				...options.headers
			},
			...options
		});
		
		if (!response.ok) {
			const error = await response.json().catch(() => ({}));
			throw new Error(error.message || `HTTP ${response.status}`);
		}
		
		return await response.json();
	} catch (err) {
		console.error(`API Error [${endpoint}]:`, err);
		throw err;
	}
}

/**
 * Fetch all business units with their aggregate metrics
 * @returns {Promise<{business_units: Array}>}
 */
export async function getBusinessUnits() {
	if (USE_MOCK_DATA) return mockGetBusinessUnits();
	return fetchAPI('/business-units');
}

/**
 * Fetch a single business unit with technician breakdown and trend data
 * @param {string} id - Business unit ID
 * @param {string} range - Time range (7d, 30d, 90d, ytd)
 * @returns {Promise<Object>}
 */
export async function getBusinessUnit(id, range = '30d') {
	if (USE_MOCK_DATA) return mockGetBusinessUnit(id, range);
	return fetchAPI(`/business-units/${encodeURIComponent(id)}?range=${range}`);
}

/**
 * Fetch a single technician with lifetime stats and trend data
 * @param {string} id - Technician ID
 * @param {string} range - Time range for trend (7d, 30d, 90d, ytd)
 * @returns {Promise<Object>}
 */
export async function getTechnician(id, range = '30d') {
	if (USE_MOCK_DATA) return mockGetTechnician(id, range);
	return fetchAPI(`/technicians/${encodeURIComponent(id)}?range=${range}`);
}

/**
 * Time range options for the range selector
 */
export const TIME_RANGES = [
	{ value: '7d', label: '7 Days' },
	{ value: '30d', label: '30 Days' },
	{ value: '90d', label: '90 Days' },
	{ value: 'ytd', label: 'Year to Date' }
];

/**
 * Metric definitions with labels, formatting, and display info
 */
export const METRIC_DEFINITIONS = {
	close_rate: {
		label: 'Close Rate',
		format: (v) => `${(v * 100).toFixed(1)}%`,
		description: 'Percentage of estimates that convert to jobs',
		higherIsBetter: true
	},
	average_ticket: {
		label: 'Avg Ticket',
		format: (v) => `$${v.toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`,
		description: 'Average revenue per completed job',
		higherIsBetter: true
	},
	total_jobs: {
		label: 'Total Jobs',
		format: (v) => v.toLocaleString('en-US'),
		description: 'Number of jobs completed',
		higherIsBetter: true
	},
	average_time_on_job: {
		label: 'Avg Time',
		format: (v) => `${Math.floor(v / 60)}h ${v % 60}m`,
		description: 'Average time spent per job',
		higherIsBetter: false  // Lower is generally more efficient
	},
	average_estimates_per_job: {
		label: 'Estimates/Job',
		format: (v) => v.toFixed(1),
		description: 'Average number of estimate options presented',
		higherIsBetter: true
	},
	callback_rate: {
		label: 'Callback Rate',
		format: (v) => `${(v * 100).toFixed(1)}%`,
		description: 'Percentage of jobs requiring a return visit',
		higherIsBetter: false  // Lower is better
	}
};

/**
 * Format a metric value using its definition
 */
export function formatMetric(key, value) {
	const def = METRIC_DEFINITIONS[key];
	if (!def) return String(value);
	return def.format(value);
}