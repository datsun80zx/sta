SELECT '=== DATA COUNTS ===' as section;

SELECT 'Jobs' as table_name, COUNT(*) as count FROM jobs
UNION ALL
SELECT 'Completed Jobs', COUNT(*) FROM jobs WHERE status = 'Completed'
UNION ALL
SELECT 'Technicians', COUNT(*) FROM technicians
UNION ALL
SELECT 'Job-Tech Links', COUNT(*) FROM job_technicians;

SELECT '=== ROLES BREAKDOWN ===' as section;
SELECT role, COUNT(*) as count FROM job_technicians GROUP BY role;

SELECT '=== BUSINESS UNITS ===' as section;
SELECT business_unit, COUNT(*) as job_count 
FROM jobs 
WHERE business_unit IS NOT NULL 
GROUP BY business_unit;

SELECT '=== SAMPLE TECHNICIAN DATA ===' as section;
SELECT t.name, jt.role, COUNT(*) as jobs
FROM technicians t
JOIN job_technicians jt ON t.id = jt.technician_id
GROUP BY t.name, jt.role
ORDER BY t.name, jt.role
LIMIT 20;

SELECT '=== PROJECT/CALLBACK FIELDS ===' as section;
SELECT 
    COUNT(*) as total_jobs,
    COUNT(project_id) as with_project,
    COUNT(warranty_for_job_id) as with_warranty_for,
    COUNT(recall_for_job_id) as with_recall_for
FROM jobs;