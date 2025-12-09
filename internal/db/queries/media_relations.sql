-- media_relations.sql
-- SQLC queries for managing parent-child relationships between media items

-- =============================================================================
-- GetMediaRelation - Retrieve a media relation by ID
-- =============================================================================
-- name: GetMediaRelation :one
SELECT * FROM media_relations
WHERE id = $1;

-- =============================================================================
-- GetMediaRelationByParentChildRelation - Find a specific relation
-- =============================================================================
-- Used to check if a relation already exists
-- name: GetMediaRelationByParentChildRelation :one
SELECT * FROM media_relations
WHERE parent_id = $1 AND child_id = $2 AND relation = $3;

-- =============================================================================
-- ListMediaRelationsByParent - Get all children of a parent
-- =============================================================================
-- Example: Get all seasons of a TV series
-- name: ListMediaRelationsByParent :many
SELECT * FROM media_relations
WHERE parent_id = $1
ORDER BY sort_index NULLS LAST, id;

-- =============================================================================
-- ListMediaRelationsByParentAndRelation - Get children of specific relation type
-- =============================================================================
-- Example: Get all episodes of a season (parent_id = season_id, relation = "season-episode")
-- name: ListMediaRelationsByParentAndRelation :many
SELECT * FROM media_relations
WHERE parent_id = $1 AND relation = $2
ORDER BY sort_index NULLS LAST, id;

-- =============================================================================
-- ListMediaRelationsByChild - Get all parents of a child
-- =============================================================================
-- Example: Find which season an episode belongs to
-- name: ListMediaRelationsByChild :many
SELECT * FROM media_relations
WHERE child_id = $1
ORDER BY created_at;

-- =============================================================================
-- ListMediaRelationsByRelation - Get all relations of a specific type
-- =============================================================================
-- Example: Get all season-episode relations
-- name: ListMediaRelationsByRelation :many
SELECT * FROM media_relations
WHERE relation = $1
ORDER BY parent_id, sort_index NULLS LAST;

-- =============================================================================
-- CreateMediaRelation - Insert a new relation
-- =============================================================================
-- name: CreateMediaRelation :one
INSERT INTO media_relations (
    parent_id,
    child_id,
    relation,
    sort_index,
    metadata
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- =============================================================================
-- UpdateMediaRelation - Update an existing relation
-- =============================================================================
-- name: UpdateMediaRelation :one
UPDATE media_relations
SET
    sort_index = COALESCE(sqlc.narg('sort_index'), sort_index),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- =============================================================================
-- UpsertMediaRelation - Insert or update a relation
-- =============================================================================
-- Atomically create or update based on parent_id + child_id + relation
-- name: UpsertMediaRelation :one
INSERT INTO media_relations (
    parent_id,
    child_id,
    relation,
    sort_index,
    metadata
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (parent_id, child_id, relation) DO UPDATE SET
    sort_index = EXCLUDED.sort_index,
    metadata = EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- =============================================================================
-- DeleteMediaRelation - Delete a relation by ID
-- =============================================================================
-- name: DeleteMediaRelation :exec
DELETE FROM media_relations
WHERE id = $1;

-- =============================================================================
-- DeleteMediaRelationByParentChildRelation - Delete a specific relation
-- =============================================================================
-- name: DeleteMediaRelationByParentChildRelation :exec
DELETE FROM media_relations
WHERE parent_id = $1 AND child_id = $2 AND relation = $3;

-- =============================================================================
-- DeleteMediaRelationsByParent - Delete all relations for a parent
-- =============================================================================
-- Example: Delete all season relations when a series is removed
-- name: DeleteMediaRelationsByParent :exec
DELETE FROM media_relations
WHERE parent_id = $1;

-- =============================================================================
-- DeleteMediaRelationsByChild - Delete all relations for a child
-- =============================================================================
-- name: DeleteMediaRelationsByChild :exec
DELETE FROM media_relations
WHERE child_id = $1;

-- =============================================================================
-- CountMediaRelationsByParent - Count children of a parent
-- =============================================================================
-- Example: Count how many episodes are in a season
-- name: CountMediaRelationsByParent :one
SELECT COUNT(*) FROM media_relations
WHERE parent_id = $1;

-- =============================================================================
-- CountMediaRelationsByParentAndRelation - Count children by relation type
-- =============================================================================
-- name: CountMediaRelationsByParentAndRelation :one
SELECT COUNT(*) FROM media_relations
WHERE parent_id = $1 AND relation = $2;
