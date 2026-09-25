import { isAxiosError } from 'axios';
import { NotApiError } from './client';

const VALIDATOR_FIELD = /Error:Field validation for '(\w+)'/g;

const FRIENDLY: [string, string][] = [
  ['invalid credentials', 'Wrong email or password.'],
  ['account temporarily locked', 'Too many attempts. Wait a few minutes and try again.'],
  ['email not verified', 'Check your inbox and confirm your email before signing in.'],
  ['email already taken', 'An account already uses this email. Try signing in instead.'],
  ['invalid or expired token', 'This link has expired. Ask for a new one.'],
  ['cannot remove the only login method', 'This is your only way to sign in, so it cannot be removed.'],
  ['this account is already linked', 'This Google account is already linked to another user.'],
  ['google sign-in is not configured', 'Google sign-in is not available right now.'],
  ['premium subscription required', 'This feature is part of Premium. Upgrade from your profile to use it.'],
  ['user not found', 'We could not find this user.'],
  ['invalid profile data', 'Some profile fields are not valid. Check them and try again.'],
  ['cannot add yourself as a friend', 'You cannot add yourself as a friend.'],
  ['already friends', 'You are already friends.'],
  ['request already pending', 'A friend request is already waiting for an answer.'],
  ['request not found', 'This friend request no longer exists.'],
  ['unauthorized action', 'You are not allowed to do that.'],
  ['access denied', 'You do not have access to this. Ask its owner for an invitation.'],
  ['event not found', 'This event no longer exists.'],
  ['track not found in this event', 'This track is no longer in the queue.'],
  ['deezer track not found', 'This track is not available on Deezer.'],
  ['invalid event parameters', 'Some event details are missing or not valid.'],
  ['vote not allowed for this event', 'Only invited guests can take part in this event.'],
  ['voting is not open now', 'Voting is closed right now. Check the event times.'],
  ['gps location required', 'Allow location access to vote at this event.'],
  ['outside the authorized voting area', 'You are too far from the event to vote.'],
  ['playlist not found', 'This playlist no longer exists.'],
  ['track not found in playlist', 'This track is no longer in the playlist.'],
  ['version conflict', 'Someone else just changed this playlist. It has been refreshed, try again.'],
  ['this track is already in the playlist', 'This track is already in the playlist.'],
  ['invalid move', 'This track cannot be moved there.'],
  ['no preview for this track', 'Deezer offers no preview for this track.'],
  ['deezer search unavailable', 'Music search is unavailable right now. Try again in a moment.'],
  ['internal error', 'Something went wrong on our side. Try again in a moment.'],
];

const BY_STATUS: Record<number, string> = {
  400: 'Something in this request is not valid.',
  401: 'Your session has expired. Sign in again.',
  403: 'You are not allowed to do that.',
  404: 'We could not find what you were looking for.',
  409: 'This conflicts with a recent change. Refresh and try again.',
  429: 'Too many attempts. Wait a moment and try again.',
};

function readable(message: string, status?: number): string {
  const fields = [...message.matchAll(VALIDATOR_FIELD)].map((m) => m[1].toLowerCase());
  if (fields.length) return `Check these fields: ${fields.join(', ')}.`;
  const lower = message.toLowerCase();
  const hit = FRIENDLY.find(([key]) => lower.startsWith(key));
  if (hit) return hit[1];
  if (status && BY_STATUS[status]) return BY_STATUS[status];
  return message;
}

export function apiError(e: unknown, fallback: string): string {
  if (e instanceof NotApiError) return 'THE SERVER URL DOES NOT POINT AT THE MUSICROOM API. CHECK IT BELOW.';
  if (isAxiosError(e)) {
    const data = e.response?.data as { message?: unknown; error?: unknown } | undefined;
    const message = data?.message ?? data?.error;
    if (typeof message === 'string' && message) return readable(message, e.response?.status).toUpperCase();
    if (!e.response) return 'CANNOT REACH THE SERVER. CHECK YOUR CONNECTION.';
    const byStatus = BY_STATUS[e.response.status];
    if (byStatus) return byStatus.toUpperCase();
  }
  return fallback;
}
