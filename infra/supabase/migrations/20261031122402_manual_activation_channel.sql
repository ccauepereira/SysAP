begin;

alter table app.activation_challenges drop constraint activation_challenges_channel_check;
alter table app.activation_challenges add constraint activation_challenges_channel_check
  check (channel in ('sms', 'email', 'manual'));

alter table app.athlete_profiles drop constraint athlete_profiles_delivery_channel_check;
alter table app.athlete_profiles add constraint athlete_profiles_delivery_channel_check
  check (enrollment_delivery_channel in ('sms', 'email', 'manual'));
alter table app.activation_invitations drop constraint activation_invitations_delivery_status_check;
alter table app.activation_invitations add constraint activation_invitations_delivery_status_check
  check (delivery_status in ('sending', 'sent', 'failed', 'unknown', 'not_applicable'));

-- Rollback requires invalidating and removing manual challenges and explicitly
-- migrating manual delivery channels/statuses to a supported representation.
-- Removing challenges irreversibly invalidates the codes already issued.
commit;
