-- name: CountAddressObjects :one
SELECT count(*)
FROM address_objects;

-- name: GetVersionInfo :one
SELECT *
FROM version_info
WHERE version_id = $1 AND file_type = $2;

-- name: VersionImported :one
SELECT EXISTS(
    SELECT 1
    FROM version_info
    WHERE version_id = $1 AND file_type = $2 AND status = 'imported'
);

-- name: ExtractionBlocker :one
SELECT version_id, status
FROM version_info
WHERE (version_id = $1 AND file_type = $2 AND status IN ('extracted', 'imported'))
   OR (version_id <> $1 AND status = 'extracted' AND file_type = $2)
ORDER BY (version_id = $1 AND file_type = $2) DESC, updated_at DESC
LIMIT 1;

-- name: UpsertVersionInfo :exec
INSERT INTO version_info (version_id, text_version, gar_xml_full_url, gar_xml_delta_url, exp_date, date, status, file_type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (version_id, file_type) DO UPDATE
SET text_version      = EXCLUDED.text_version,
    gar_xml_full_url  = COALESCE(EXCLUDED.gar_xml_full_url, version_info.gar_xml_full_url),
    gar_xml_delta_url = COALESCE(EXCLUDED.gar_xml_delta_url, version_info.gar_xml_delta_url),
    exp_date          = COALESCE(EXCLUDED.exp_date, version_info.exp_date),
    date              = EXCLUDED.date,
    status            = EXCLUDED.status,
    file_type         = EXCLUDED.file_type,
    updated_at        = now();

-- name: MarkVersionImported :execrows
UPDATE version_info
SET status = 'imported', updated_at = now()
WHERE version_id = $1 AND file_type = $2;
