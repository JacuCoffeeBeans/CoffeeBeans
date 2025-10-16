ALTER TABLE public.profiles
ADD COLUMN stripe_account_id TEXT UNIQUE,
ADD COLUMN stripe_account_status TEXT DEFAULT 'restricted';
