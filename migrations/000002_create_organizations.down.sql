DROP TRIGGER IF EXISTS set_organizations_updated_at ON organizations;
DROP TABLE IF EXISTS organizations;
-- Keep trigger_set_updated_at function; other tables may use it.
