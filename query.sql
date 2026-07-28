select pol.polname, pol.polcmd, pol.polqual, pol.polwithcheck
from pg_policy pol
join pg_class c on pol.polrelid = c.oid
where c.relname = 'athlete_profiles';
