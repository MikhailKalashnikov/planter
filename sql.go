package main

const columDefSQL = `
SELECT
    a.attnum AS field_ordinal,
    a.attname AS column_name,
    pd.description AS description,
    replace(UPPER(format_type(a.atttypid, a.atttypmod)), 'TIMESTAMP WITH TIME ZONE', 'TIMESTAMPTZ') AS data_type,
    a.attnotnull AS not_null,
    COALESCE(ct.contype = 'p', false) AS  is_primary_key,
    COALESCE(ct2.contype = 'u', false) AS  is_unique,
    replace(translate(pg_get_expr(adbin, adrelid), '()', ''), '::timestamp with time zone', '') AS def_val
FROM pg_attribute a
JOIN ONLY pg_class c ON c.oid = a.attrelid
JOIN ONLY pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_constraint ct ON ct.conrelid = c.oid AND a.attnum = ANY(ct.conkey) AND ct.contype IN ('p' )
LEFT JOIN pg_constraint ct2 ON ct2.conrelid = c.oid AND a.attnum = ANY(ct2.conkey) AND ct2.contype IN ('u')
LEFT JOIN pg_attrdef ad ON ad.adrelid = c.oid AND ad.adnum = a.attnum
LEFT JOIN pg_description pd ON pd.objoid = a.attrelid AND pd.objsubid = a.attnum
WHERE a.attisdropped = false
AND n.nspname = $1
AND c.relname = $2
AND a.attnum > 0
ORDER BY a.attnum
`

const tableDefSQL = `
SELECT
  c.relname AS table_name,
  pd.description AS description
FROM pg_class c
JOIN ONLY pg_namespace n
ON n.oid = c.relnamespace
LEFT JOIN pg_description pd ON pd.objoid = c.oid AND pd.objsubid = 0
WHERE n.nspname = $1
AND c.relkind = 'r'
ORDER BY c.relname
`

const fkDefSQL = `
select
 DISTINCT cl.relname as "parent_table"
  , con.conname
  , con.nspname "conn_schema"
  , ns.nspname "parent_schema"
from (
  select
    unnest(con1.conkey) as "parent"
    , unnest(con1.confkey) as "child"
    , con1.confrelid
    , con1.conrelid
    , con1.conname
    , ns.nspname
  from pg_class cl
  join pg_namespace ns on cl.relnamespace = ns.oid
  join pg_constraint con1 on con1.conrelid = cl.oid
  where ns.nspname = $1
  and cl.relname = $2
  and con1.contype = 'f'
) con
join pg_attribute att on att.attrelid = con.confrelid and att.attnum = con.child
join pg_class cl on cl.oid = con.confrelid
join pg_namespace ns on cl.relnamespace = ns.oid
order by con.conname
`
