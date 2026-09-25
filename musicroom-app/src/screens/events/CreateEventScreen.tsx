import React, { useEffect, useState } from 'react';
import { View, StyleSheet, Pressable } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { eventsApi, EventLicense, CreateEventInput } from '../../api/events';
import { friendsApi } from '../../api/friends';
import { User } from '../../api/users';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';
import { colors, spacing } from '../../theme';
import { apiError } from '../../api/error';
import {
  currentPosition,
  describePlace,
  findPlace,
  formatCoords,
  formatWindow,
  parseLocalInput,
  toLocalInput,
} from './eventFormat';

const LICENSES: EventLicense[] = ['open', 'invited_only', 'time_and_location'];
const LICENSE_LABELS: Record<EventLicense, string> = {
  open: 'OPEN',
  invited_only: 'INVITE ONLY',
  time_and_location: 'TIME + LOCATION',
};
const LICENSE_HINTS: Record<EventLicense, string> = {
  open: '// ANYONE WHO CAN SEE THE EVENT CAN SUGGEST AND VOTE',
  invited_only: '// ONLY YOU AND THE FRIENDS YOU INVITE CAN SUGGEST AND VOTE',
  time_and_location: '// ONLY PEOPLE AT THE PLACE, DURING THE HOURS BELOW, CAN VOTE',
};

const RADII = [100, 500, 1000, 5000];
const DURATIONS = [1, 3, 6, 24];

type Place = { latitude: number; longitude: number; label: string | null };

function nextQuarter(): Date {
  const d = new Date();
  d.setSeconds(0, 0);
  d.setMinutes(Math.ceil((d.getMinutes() + 1) / 15) * 15);
  return d;
}

export default function CreateEventScreen() {
  const nav = useNavigation<any>();
  const [name, setName] = useState('');
  const [desc, setDesc] = useState('');
  const [license, setLicense] = useState<EventLicense>('open');
  const [visibility, setVisibility] = useState<'public' | 'private'>('public');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const [place, setPlace] = useState<Place | null>(null);
  const [address, setAddress] = useState('');
  const [locating, setLocating] = useState(false);
  const [placeError, setPlaceError] = useState('');
  const [radius, setRadius] = useState(500);

  const [startText, setStartText] = useState(() => toLocalInput(nextQuarter()));
  const [hours, setHours] = useState(3);
  const start = parseLocalInput(startText);
  const end = start ? new Date(start.getTime() + hours * 3600_000) : null;

  const [friends, setFriends] = useState<User[]>([]);
  const [invites, setInvites] = useState<Set<string>>(new Set());
  const [friendsError, setFriendsError] = useState('');
  const needsGuests = visibility === 'private' || license === 'invited_only';

  useEffect(() => {
    friendsApi.list().then(setFriends, (e) => setFriendsError(apiError(e, 'COULD NOT LOAD YOUR FRIENDS')));
  }, []);

  async function useMyLocation() {
    setPlaceError('');
    setLocating(true);
    try {
      const pos = await currentPosition();
      setPlace({ ...pos, label: await describePlace(pos.latitude, pos.longitude) });
    } catch (e: any) {
      setPlaceError(e?.message ?? 'COULD NOT FIND WHERE YOU ARE.');
    } finally {
      setLocating(false);
    }
  }

  async function findAddress() {
    if (!address.trim()) return;
    setPlaceError('');
    setLocating(true);
    try {
      const hit = await findPlace(address);
      if (!hit) {
        setPlaceError('NO PLACE MATCHES THIS ADDRESS. TRY ADDING THE CITY.');
        return;
      }
      const label = (await describePlace(hit.latitude, hit.longitude)) ?? address.trim();
      setPlace({ latitude: hit.latitude, longitude: hit.longitude, label });
    } catch {
      setPlaceError('ADDRESS SEARCH IS UNAVAILABLE RIGHT NOW. TRY AGAIN, OR USE YOUR CURRENT LOCATION.');
    } finally {
      setLocating(false);
    }
  }

  function toggleInvite(id: string) {
    setInvites((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function validate(): string | null {
    if (!name.trim()) return 'GIVE YOUR EVENT A NAME.';
    if (license !== 'time_and_location') return null;
    if (!place) return 'PICK WHERE THE EVENT TAKES PLACE.';
    if (!start) return 'THE START TIME MUST LOOK LIKE 2026-10-12 20:30.';
    if (end && end.getTime() < Date.now()) return 'THIS EVENT WOULD ALREADY BE OVER. PICK A LATER START.';
    return null;
  }

  async function create() {
    const problem = validate();
    if (problem) {
      setError(problem);
      return;
    }
    setError('');
    setLoading(true);
    try {
      const body: CreateEventInput = { name: name.trim(), description: desc, visibility, license };
      if (license === 'time_and_location' && place && start && end) {
        body.vote_latitude = place.latitude;
        body.vote_longitude = place.longitude;
        body.vote_radius_meters = radius;
        body.vote_start_at = start.toISOString();
        body.vote_end_at = end.toISOString();
      }
      const event = await eventsApi.create(body);
      if (needsGuests) await Promise.allSettled([...invites].map((id) => eventsApi.invite(event.id, id)));
      nav.replace('EventDetail', { eventId: event.id });
    } catch (e: any) {
      setError(apiError(e, 'COULD NOT CREATE IT. TRY AGAIN.'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Screen title="NEW EVENT" scroll>
      <Input label="NAME" value={name} onChangeText={setName} />
      <Input label="DESCRIPTION (OPTIONAL)" value={desc} onChangeText={setDesc} multiline />

      <View style={styles.divider} />
      <Text size="sm" dim>{'// VISIBILITY'}</Text>
      <View style={styles.row}>
        <Button label="PUBLIC" onPress={() => setVisibility('public')} variant={visibility === 'public' ? 'primary' : 'ghost'} />
        <Button label="PRIVATE" onPress={() => setVisibility('private')} variant={visibility === 'private' ? 'primary' : 'ghost'} />
      </View>
      <Text size="xs" dim>
        {visibility === 'public'
          ? '// EVERYONE SEES THIS EVENT IN THEIR LIST'
          : '// ONLY YOU AND THE FRIENDS YOU INVITE SEE THIS EVENT'}
      </Text>

      <View style={styles.divider} />
      <Text size="sm" dim>{'// WHO CAN VOTE'}</Text>
      {LICENSES.map((l) => (
        <Button key={l} label={LICENSE_LABELS[l]} onPress={() => setLicense(l)} variant={license === l ? 'primary' : 'ghost'} />
      ))}
      <Text size="xs" dim>{LICENSE_HINTS[license]}</Text>

      {license === 'time_and_location' && (
        <>
          <View style={styles.divider} />
          <Text size="sm" dim>{'// WHERE'}</Text>
          <Button label={locating ? 'LOCATING...' : '◎ USE MY CURRENT LOCATION'} onPress={useMyLocation} disabled={locating} />
          <View style={styles.addressRow}>
            <View style={{ flex: 1 }}>
              <Input label="OR AN ADDRESS" value={address} onChangeText={setAddress} onSubmitEditing={findAddress} />
            </View>
            <Button label="FIND" onPress={findAddress} variant="ghost" disabled={locating} />
          </View>
          {placeError ? <Text size="sm" danger>{`! ${placeError}`}</Text> : null}
          {place ? (
            <View style={styles.summary}>
              <Text size="sm" accent>{`> ${place.label ?? 'PICKED POSITION'}`}</Text>
              <Text size="xs" dim>{`// ${formatCoords(place.latitude, place.longitude)}`}</Text>
            </View>
          ) : (
            <Text size="xs" dim>{'// NO PLACE PICKED YET'}</Text>
          )}

          <Text size="sm" dim style={styles.subhead}>{'// VOTING AREA AROUND IT'}</Text>
          <View style={styles.row}>
            {RADII.map((r) => (
              <Button
                key={r}
                label={r < 1000 ? `${r} M` : `${r / 1000} KM`}
                onPress={() => setRadius(r)}
                variant={radius === r ? 'primary' : 'ghost'}
              />
            ))}
          </View>

          <View style={styles.divider} />
          <Text size="sm" dim>{'// WHEN'}</Text>
          <View style={styles.addressRow}>
            <View style={{ flex: 1 }}>
              <Input label="STARTS (YYYY-MM-DD HH:MM, YOUR TIME)" value={startText} onChangeText={setStartText} autoCapitalize="none" />
            </View>
            <Button label="NOW" onPress={() => setStartText(toLocalInput(new Date()))} variant="ghost" />
          </View>
          <Text size="sm" dim style={styles.subhead}>{'// LASTS'}</Text>
          <View style={styles.row}>
            {DURATIONS.map((h) => (
              <Button key={h} label={`${h} H`} onPress={() => setHours(h)} variant={hours === h ? 'primary' : 'ghost'} />
            ))}
          </View>
          {start && end ? (
            <Text size="sm" accent style={styles.summary}>{`> ${formatWindow(start.toISOString(), end.toISOString())}`}</Text>
          ) : (
            <Text size="sm" danger>{'! THE START TIME MUST LOOK LIKE 2026-10-12 20:30'}</Text>
          )}
        </>
      )}

      {needsGuests && (
        <>
          <View style={styles.divider} />
          <Text size="sm" dim>{'// INVITE FRIENDS'}</Text>
          {friendsError ? (
            <Text size="sm" danger>{`! ${friendsError}`}</Text>
          ) : friends.length === 0 ? (
            <Text dim size="xs">{'// NO FRIENDS TO INVITE YET. ADD SOME FROM THE FRIENDS TAB.'}</Text>
          ) : (
            friends.map((f) => {
              const on = invites.has(f.id);
              return (
                <Pressable
                  key={f.id}
                  onPress={() => toggleInvite(f.id)}
                  style={styles.friendRow}
                  accessibilityRole="checkbox"
                  accessibilityState={{ checked: on }}
                  accessibilityLabel={`Invite ${f.display_name}`}
                >
                  <Text size="sm" style={{ flex: 1 }} dim={!on}>{f.display_name}</Text>
                  <Text size="xs" accent={on} dim={!on}>{on ? '[INVITED]' : ' INVITE '}</Text>
                </Pressable>
              );
            })
          )}
        </>
      )}

      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
      <View style={{ height: spacing.md }} />
      <Button label={loading ? 'CREATING...' : 'CREATE EVENT'} onPress={create} disabled={loading} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  divider: { height: 1, backgroundColor: colors.border, marginVertical: spacing.sm },
  row: { flexDirection: 'row', gap: 8, flexWrap: 'wrap' },
  addressRow: { flexDirection: 'row', alignItems: 'flex-end', gap: 8 },
  subhead: { marginTop: spacing.sm },
  summary: { marginVertical: spacing.xs },
  friendRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
});
