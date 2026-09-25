import React, { useEffect, useState, useRef } from 'react';
import { FlatList, View, Pressable, StyleSheet, TextInput, Modal } from 'react-native';
import { useRoute, RouteProp } from '@react-navigation/native';
import { eventsApi, MusicEvent, EventTrack } from '../../api/events';
import { tracksApi, Track } from '../../api/tracks';
import { friendsApi } from '../../api/friends';
import { User } from '../../api/users';
import { apiError } from '../../api/error';
import { useAuth } from '../../context/AuthContext';
import { useWebSocket } from '../../ws/useWebSocket';
import { usePreviewPlayer } from '../../audio/usePreviewPlayer';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Button from '../../components/ui/Button';
import { colors, spacing, fonts, fontSize } from '../../theme';
import {
  currentPosition,
  describePlace,
  distanceMeters,
  formatCoords,
  formatDistance,
  formatWindow,
  voteWindow,
} from './eventFormat';

type Route = RouteProp<{ EventDetail: { eventId: string } }, 'EventDetail'>;

const LICENSE_TEXT: Record<MusicEvent['license'], string> = {
  open: 'ANYONE WHO CAN SEE THIS EVENT CAN SUGGEST AND VOTE',
  invited_only: 'ONLY THE HOST AND INVITED GUESTS CAN SUGGEST AND VOTE',
  time_and_location: 'ONLY PEOPLE AT THE PLACE, DURING THE HOURS BELOW, CAN VOTE',
};

type Presence =
  | { state: 'checking' }
  | { state: 'near' | 'far'; distance: number }
  | { state: 'unknown'; reason: string };

export default function EventDetailScreen() {
  const route = useRoute<Route>();
  const { eventId } = route.params;
  const { user } = useAuth();

  const [event, setEvent] = useState<MusicEvent | null>(null);
  const [tracks, setTracks] = useState<EventTrack[]>([]);
  const [searchQ, setSearchQ] = useState('');
  const [searchResults, setSearchResults] = useState<Track[]>([]);
  const [showSearch, setShowSearch] = useState(false);
  const [error, setError] = useState('');
  const [searchError, setSearchError] = useState('');
  const [loadFailed, setLoadFailed] = useState(false);
  const queueReqRef = useRef(0);

  const [placeLabel, setPlaceLabel] = useState<string | null>(null);
  const [presence, setPresence] = useState<Presence | null>(null);

  const [guests, setGuests] = useState<{ id: string; display_name: string }[]>([]);
  const [friends, setFriends] = useState<User[]>([]);
  const [hostError, setHostError] = useState('');

  const isHost = !!event && event.owner_id === user?.id;
  const geo =
    event?.license === 'time_and_location' &&
    event.vote_latitude != null &&
    event.vote_longitude != null &&
    event.vote_radius_meters != null
      ? { lat: event.vote_latitude, lng: event.vote_longitude, radius: event.vote_radius_meters }
      : null;

  const tracksRef = useRef<EventTrack[]>([]);
  tracksRef.current = tracks;
  const player = usePreviewPlayer((finishedId) => {
    const list = tracksRef.current;
    const next = list[list.findIndex((t) => t.id === finishedId) + 1];
    if (next) player.play(next);
  });

  useEffect(() => {
    load();
  }, [eventId]);

  useEffect(() => {
    if (!geo) return;
    describePlace(geo.lat, geo.lng).then(setPlaceLabel);
    locate();
  }, [geo?.lat, geo?.lng, geo?.radius]);

  useEffect(() => {
    if (isHost) loadHostPanel();
  }, [isHost]);

  useWebSocket('events', eventId, (msg) => {
    if (msg.type === 'queue:updated') loadTracks();
  });

  async function load() {
    setError('');
    const req = ++queueReqRef.current;
    try {
      const ev = await eventsApi.get(eventId);
      setEvent(ev);
      if (req === queueReqRef.current) setTracks(ev.queue ?? []);
      setLoadFailed(false);
    } catch (e) {
      setLoadFailed(true);
      setError(apiError(e, 'COULD NOT LOAD THIS EVENT'));
    }
  }

  async function loadTracks() {
    const req = ++queueReqRef.current;
    try {
      const queue = await eventsApi.tracks(eventId);
      if (req !== queueReqRef.current) return;
      setTracks(queue);
      setError('');
    } catch (e) {
      if (req !== queueReqRef.current) return;
      setError(apiError(e, 'COULD NOT REFRESH THE QUEUE'));
    }
  }

  async function locate() {
    if (!geo) return;
    setPresence({ state: 'checking' });
    try {
      const pos = await currentPosition();
      const d = distanceMeters(pos.latitude, pos.longitude, geo.lat, geo.lng);
      setPresence({ state: d <= geo.radius ? 'near' : 'far', distance: d });
    } catch (e: any) {
      setPresence({ state: 'unknown', reason: e?.message ?? 'COULD NOT FIND WHERE YOU ARE.' });
    }
  }

  async function loadHostPanel() {
    setHostError('');
    try {
      const [invited, mine] = await Promise.all([eventsApi.invitations(eventId), friendsApi.list()]);
      setGuests(invited);
      setFriends(mine);
    } catch (e) {
      setHostError(apiError(e, 'COULD NOT LOAD YOUR GUEST LIST'));
    }
  }

  async function invite(friend: User) {
    setHostError('');
    try {
      await eventsApi.invite(eventId, friend.id);
    } catch (e) {
      setHostError(apiError(e, `COULD NOT INVITE ${friend.display_name.toUpperCase()}`));
      return;
    }
    loadHostPanel();
  }

  async function vote(trackId: string) {
    setError('');
    try {
      let coords: { latitude: number; longitude: number } | undefined;
      if (event?.license === 'time_and_location') coords = await currentPosition();
      await eventsApi.vote(eventId, trackId, coords);
    } catch (e) {
      setError(apiError(e, (e as Error)?.message || 'YOUR VOTE DID NOT GO THROUGH. TRY AGAIN.'));
      return;
    }
    loadTracks();
  }

  async function suggest(track: Track) {
    setSearchError('');
    try {
      await eventsApi.suggestTrack(eventId, track.deezer_id);
    } catch (e) {
      setSearchError(apiError(e, 'COULD NOT SUGGEST THIS TRACK'));
      return;
    }
    setShowSearch(false);
    setSearchResults([]);
    setSearchQ('');
    loadTracks();
  }

  async function search() {
    if (!searchQ.trim()) return;
    setSearchError('');
    try {
      setSearchResults(await tracksApi.search(searchQ));
    } catch (e) {
      setSearchResults([]);
      setSearchError(apiError(e, 'SEARCH FAILED. TRY AGAIN.'));
    }
  }

  if (!event && loadFailed) {
    return (
      <Screen title="EVENT">
        <Text size="sm" danger>{`! ${error}`}</Text>
        <View style={{ height: spacing.md }} />
        <Button label="RETRY" onPress={load} />
      </Screen>
    );
  }
  if (!event) return <Screen title="LOADING..."><Text dim>{'// FETCHING...'}</Text></Screen>;

  const where = geo ? placeLabel ?? formatCoords(geo.lat, geo.lng) : null;
  const phase = voteWindow(event);

  if (geo && !isHost && presence?.state !== 'near') {
    return (
      <Screen title={event.name}>
        <Text size="sm" dim>{`// ${where}`}</Text>
        {event.vote_start_at && event.vote_end_at ? (
          <Text size="sm" dim>{`// ${formatWindow(event.vote_start_at, event.vote_end_at)}`}</Text>
        ) : null}
        <View style={{ height: spacing.md }} />
        {!presence || presence.state === 'checking' ? (
          <Text accent>{'// CHECKING WHERE YOU ARE...'}</Text>
        ) : presence.state === 'far' ? (
          <Text danger>
            {`! YOU ARE ${formatDistance(presence.distance)} AWAY. THIS EVENT OPENS WITHIN ${formatDistance(geo.radius)} OF THE PLACE ABOVE.`}
          </Text>
        ) : presence.state === 'unknown' ? (
          <Text danger>{`! ${presence.reason} THIS EVENT ONLY OPENS TO PEOPLE AT THE PLACE.`}</Text>
        ) : null}
        <View style={{ height: spacing.md }} />
        <Button label="CHECK AGAIN" onPress={locate} disabled={presence?.state === 'checking'} />
      </Screen>
    );
  }

  const invitedIds = new Set(guests.map((g) => g.id));
  const invitable = friends.filter((f) => !invitedIds.has(f.id));

  const header = (
    <View>
      <View style={styles.info}>
        {isHost ? (
          <Text size="sm" accent>{'// YOU ARE THE HOST'}</Text>
        ) : (
          <Text size="sm" dim>{`// HOSTED BY ${event.owner_display_name ?? 'SOMEONE'}`}</Text>
        )}
        {event.description ? <Text size="sm">{event.description}</Text> : null}
        <Text size="xs" dim>{`// ${event.visibility === 'private' ? 'PRIVATE — INVITED GUESTS ONLY' : 'PUBLIC'}`}</Text>
        <Text size="xs" dim>{`// ${LICENSE_TEXT[event.license]}`}</Text>
        {event.vote_start_at && event.vote_end_at ? (
          <Text size="sm">
            {`WHEN  ${formatWindow(event.vote_start_at, event.vote_end_at)}${
              phase === 'open' ? '  [LIVE NOW]' : phase === 'closed' ? '  [OVER]' : '  [NOT STARTED]'
            }`}
          </Text>
        ) : null}
        {geo ? (
          <>
            <Text size="sm">{`WHERE ${where}`}</Text>
            <Text size="xs" dim>{`// VOTES COUNT WITHIN ${formatDistance(geo.radius)} OF IT`}</Text>
            {presence?.state === 'near' || presence?.state === 'far' ? (
              <Text size="xs" accent={presence.state === 'near'} danger={presence.state === 'far'}>
                {presence.state === 'near'
                  ? `// YOU ARE HERE (${formatDistance(presence.distance)} AWAY)`
                  : `// YOU ARE ${formatDistance(presence.distance)} AWAY — TOO FAR TO VOTE`}
              </Text>
            ) : presence?.state === 'unknown' ? (
              <Text size="xs" danger>{`// ${presence.reason}`}</Text>
            ) : null}
          </>
        ) : null}
      </View>

      {isHost && (
        <View style={styles.info}>
          <Text size="sm" dim>{'// GUESTS'}</Text>
          {hostError ? <Text size="sm" danger>{`! ${hostError}`}</Text> : null}
          {guests.length === 0 ? (
            <Text size="xs" dim>{'// NOBODY INVITED YET'}</Text>
          ) : (
            <Text size="sm">{guests.map((g) => g.display_name).join(', ')}</Text>
          )}
          {invitable.length > 0 && <Text size="xs" dim style={styles.subhead}>{'// INVITE A FRIEND'}</Text>}
          {invitable.map((f) => (
            <Pressable
              key={f.id}
              onPress={() => invite(f)}
              style={styles.friendRow}
              accessibilityRole="button"
              accessibilityLabel={`Invite ${f.display_name}`}
            >
              <Text size="sm" style={{ flex: 1 }}>{f.display_name}</Text>
              <Text size="xs" accent>{'+ INVITE'}</Text>
            </Pressable>
          ))}
          {friends.length === 0 && !hostError ? (
            <Text size="xs" dim>{'// ADD FRIENDS FROM THE FRIENDS TAB TO INVITE THEM HERE'}</Text>
          ) : null}
        </View>
      )}

      <View style={styles.info}>
        {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
        {player.error ? <Text size="sm" danger>{`! ${player.error}`}</Text> : null}
        <Button label="+ SUGGEST TRACK" onPress={() => { setSearchError(''); setShowSearch(true); }} />
        {tracks.length > 0 && (
          <Text size="xs" dim>{'// PREVIEWS ARE THE 30-SECOND EXTRACTS DEEZER SHARES'}</Text>
        )}
      </View>
    </View>
  );

  return (
    <Screen title={event.name} padded={false}>
      <FlatList
        data={tracks}
        keyExtractor={(t) => t.id}
        ListHeaderComponent={header}
        contentContainerStyle={styles.list}
        ListEmptyComponent={<Text dim style={styles.empty}>{'// NO TRACKS YET. SUGGEST THE FIRST ONE.'}</Text>}
        renderItem={({ item, index }) => (
          <View style={styles.track}>
            <View style={styles.trackInfo}>
              <Text size="xs" dim>{`${String(index + 1).padStart(2, '0')}. `}</Text>
              <View style={{ flex: 1 }}>
                <Text size="sm" accent={player.playingId === item.id}>{item.title}</Text>
                <Text size="xs" dim>{item.artist_name}</Text>
              </View>
              <Text size="md" accent>{`×${item.vote_count}`}</Text>
            </View>
            <View style={styles.trackActions}>
              <Button
                label={item.voted ? 'VOTED' : 'VOTE'}
                onPress={() => vote(item.id)}
                variant={item.voted ? 'ghost' : 'primary'}
                disabled={item.voted}
              />
              <Button
                label={
                  player.loadingId === item.id ? '...' : player.playingId === item.id ? '■ STOP' : '▶ PREVIEW'
                }
                onPress={() => player.toggle(item)}
                variant="ghost"
              />
            </View>
          </View>
        )}
      />

      <Modal visible={showSearch} animationType="slide" onRequestClose={() => setShowSearch(false)}>
        <View style={styles.modal}>
          <Text size="md" accent>{'■ SEARCH TRACKS'}</Text>
          {searchError ? <Text size="sm" danger>{`! ${searchError}`}</Text> : null}
          <View style={styles.searchRow}>
            <TextInput
              style={styles.searchInput}
              placeholder="ARTIST / TITLE..."
              placeholderTextColor={colors.textDim}
              value={searchQ}
              onChangeText={setSearchQ}
              onSubmitEditing={search}
              returnKeyType="search"
            />
            <Pressable onPress={search} style={styles.searchBtn}>
              <Text accent>{'SEARCH'}</Text>
            </Pressable>
          </View>
          <FlatList
            data={searchResults}
            keyExtractor={(t) => String(t.deezer_id)}
            renderItem={({ item }) => (
              <Pressable style={styles.result} onPress={() => suggest(item)}>
                <Text size="sm" accent>{'> '}</Text>
                <View>
                  <Text size="sm">{item.title}</Text>
                  <Text size="xs" dim>{item.artist_name}</Text>
                </View>
              </Pressable>
            )}
          />
          <Button label="CLOSE" onPress={() => setShowSearch(false)} variant="ghost" />
        </View>
      </Modal>
    </Screen>
  );
}

const styles = StyleSheet.create({
  info: { padding: spacing.md, gap: 4, borderBottomWidth: 1, borderBottomColor: colors.border },
  subhead: { marginTop: spacing.sm },
  friendRow: { flexDirection: 'row', alignItems: 'center', paddingVertical: spacing.xs },
  list: { paddingBottom: spacing.md },
  empty: { margin: spacing.md },
  track: {
    marginVertical: spacing.xs,
    marginHorizontal: spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    paddingBottom: spacing.xs,
  },
  trackInfo: { flexDirection: 'row', alignItems: 'center' },
  trackActions: { flexDirection: 'row', gap: 8, marginTop: spacing.xs },
  modal: { flex: 1, backgroundColor: colors.bg, padding: spacing.md },
  searchRow: { flexDirection: 'row', gap: 8, marginVertical: spacing.sm },
  searchInput: {
    flex: 1,
    borderWidth: 1,
    borderColor: colors.accent,
    backgroundColor: colors.surface,
    color: colors.text,
    fontFamily: fonts.mono,
    fontSize: fontSize.md,
    padding: spacing.sm,
  },
  searchBtn: { justifyContent: 'center', padding: spacing.sm, borderWidth: 1, borderColor: colors.accentDim },
  result: { flexDirection: 'row', alignItems: 'center', paddingVertical: spacing.sm, borderBottomWidth: 1, borderBottomColor: colors.border },
});
