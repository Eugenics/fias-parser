-- name: CountAddressObjects :one
SELECT count(*)
FROM address_objects;

-- name: GetVersionInfo :one
SELECT *
FROM version_info
WHERE version_id = $1;

-- name: VersionImported :one
SELECT EXISTS(
    SELECT 1
    FROM version_info
    WHERE version_id = $1 AND status = 'imported'
);

-- name: UpsertVersionInfo :exec
INSERT INTO version_info (version_id, text_version, gar_xml_full_url, gar_xml_delta_url, exp_date, date, status, file_type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (version_id) DO UPDATE
SET text_version      = EXCLUDED.text_version,
    gar_xml_full_url  = EXCLUDED.gar_xml_full_url,
    gar_xml_delta_url = EXCLUDED.gar_xml_delta_url,
    exp_date          = EXCLUDED.exp_date,
    date              = EXCLUDED.date,
    status            = EXCLUDED.status,
    file_type         = EXCLUDED.file_type,
    updated_at        = now();
