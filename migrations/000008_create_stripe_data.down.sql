DROP TABLE IF EXISTS stripe_payments;
DROP TRIGGER IF EXISTS set_stripe_subscriptions_updated_at ON stripe_subscriptions;
DROP TABLE IF EXISTS stripe_subscriptions;
