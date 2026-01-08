/**
 * Mock data for development and testing
 * 
 * This file provides realistic test data that matches the API contract.
 * You can import these functions in place of the real API calls during development.
 */

// Helper to generate random metrics
function generateMetrics(baseMultiplier = 1) {
	return {
		close_rate: 0.65 + Math.random() * 0.2,
		average_ticket: Math.round((400 + Math.random() * 200) * baseMultiplier),
		total_jobs: Math.round((100 + Math.random() * 100) * baseMultiplier),
		average_time_on_job: Math.round(60 + Math.random() * 60),
		average_estimates_per_job: 1.5 + Math.random() * 1.5,
		callback_rate: 0.03 + Math.random() * 0.07
	};
}

// Helper to generate trend data
function generateTrend(days = 30) {
	const trend = [];
	const today = new Date();
	
	for (let i = days; i >= 0; i--) {
		const date = new Date(today);
		date.setDate(date.getDate() - i);
		
		trend.push({
			date: date.toISOString().split('T')[0],
			metrics: generateMetrics()
		});
	}
	
	return trend;
}

// Mock business units
export const mockBusinessUnits = {
	business_units: [
		{
			id: 'plumbing',
			name: 'Plumbing',
			metrics: generateMetrics(1.2)
		},
		{
			id: 'hvac',
			name: 'HVAC',
			metrics: generateMetrics(1.5)
		},
		{
			id: 'electrical',
			name: 'Electrical',
			metrics: generateMetrics(1.0)
		},
		{
			id: 'drains',
			name: 'Drains',
			metrics: generateMetrics(0.8)
		}
	]
};

// Mock technicians for each business unit
const techniciansByUnit = {
	plumbing: [
		{ id: 'tech-001', name: 'Mike Thompson' },
		{ id: 'tech-002', name: 'Sarah Chen' },
		{ id: 'tech-003', name: 'James Wilson' },
		{ id: 'tech-004', name: 'Maria Garcia' },
		{ id: 'tech-005', name: 'David Brown' }
	],
	hvac: [
		{ id: 'tech-006', name: 'Chris Martinez' },
		{ id: 'tech-007', name: 'Ashley Johnson' },
		{ id: 'tech-008', name: 'Ryan Lee' },
		{ id: 'tech-009', name: 'Nicole Davis' }
	],
	electrical: [
		{ id: 'tech-010', name: 'Kevin Williams' },
		{ id: 'tech-011', name: 'Amanda Taylor' },
		{ id: 'tech-012', name: 'Brandon Moore' }
	],
	drains: [
		{ id: 'tech-013', name: 'Tyler Anderson' },
		{ id: 'tech-014', name: 'Jessica White' }
	]
};

// Generate mock data for a specific business unit
export function getMockBusinessUnit(id, range = '30d') {
	const unit = mockBusinessUnits.business_units.find(u => u.id === id);
	if (!unit) return null;
	
	const days = range === '7d' ? 7 : range === '30d' ? 30 : range === '90d' ? 90 : 180;
	const techs = techniciansByUnit[id] || [];
	
	return {
		id: unit.id,
		name: unit.name,
		metrics: generateMetrics(1.0),
		technicians: techs.map(t => ({
			...t,
			metrics: generateMetrics()
		})),
		trend: generateTrend(days)
	};
}

// Generate mock data for a specific technician
export function getMockTechnician(id, range = '30d') {
	// Find which unit this tech belongs to
	let techData = null;
	let unitData = null;
	
	for (const [unitId, techs] of Object.entries(techniciansByUnit)) {
		const found = techs.find(t => t.id === id);
		if (found) {
			techData = found;
			unitData = mockBusinessUnits.business_units.find(u => u.id === unitId);
			break;
		}
	}
	
	if (!techData || !unitData) return null;
	
	const days = range === '7d' ? 7 : range === '30d' ? 30 : range === '90d' ? 90 : 180;
	
	return {
		id: techData.id,
		name: techData.name,
		business_unit: {
			id: unitData.id,
			name: unitData.name
		},
		lifetime_metrics: generateMetrics(1.2),
		trend: generateTrend(days)
	};
}

/**
 * Mock API functions - drop-in replacements for the real API
 * 
 * Usage: In api.js, you can temporarily replace the real functions:
 * 
 * import { mockBusinessUnits, getMockBusinessUnit, getMockTechnician } from './mock-data.js';
 * 
 * export async function getBusinessUnits() {
 *   await delay(500); // Simulate network latency
 *   return mockBusinessUnits;
 * }
 */

function delay(ms) {
	return new Promise(resolve => setTimeout(resolve, ms));
}

export async function mockGetBusinessUnits() {
	await delay(300);
	return mockBusinessUnits;
}

export async function mockGetBusinessUnit(id, range) {
	await delay(400);
	const data = getMockBusinessUnit(id, range);
	if (!data) throw new Error('Business unit not found');
	return data;
}

export async function mockGetTechnician(id, range) {
	await delay(400);
	const data = getMockTechnician(id, range);
	if (!data) throw new Error('Technician not found');
	return data;
}