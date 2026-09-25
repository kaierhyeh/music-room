import React, { useCallback, useState, useRef } from 'react';
import { View, StyleSheet, TextInput, Modal, Pressable, FlatList, Platform } from 'react-native';
import { useRoute, RouteProp, useNavigation, useFocusEffect } from '@react-navigation/native';
import DraggableFlatList, { RenderItemParams, ScaleDecorator } from 'react-native-draggable-flatlist';
import { playlistsApi, Playlist, PlaylistTrack } from '../../api/playlists';
import { tracksApi, Track } from '../../api/tracks';
import { apiError } from '../../api/error';
import { useAuth } from '../../context/AuthContext';
import { useWebSocket } from '../../ws/useWebSocket';
import { usePreviewPlayer } from '../../audio/usePreviewPlayer';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Button from '../../components/ui/Button';
import PlaylistByline from '../../components/PlaylistByline';
import { colors, spacing, fonts, fontSize } from '../../theme';

type Route = RouteProp<{ PlaylistDetail: { playlistId: string } }, 'PlaylistDetail'>;

const MOVE_MAX_RETRIES = 5;
const MOVE_BACKOFF_BASE_MS = 20;

function backoff(attempt: number): Promise<void> {
  const span = MOVE_BACKOFF_BASE_MS * 2 ** attempt;
  return new Promise((resolve) => setTimeout(resolve, span + Math.random() * span * 1.5));
}

const CAN_DRAG = Platform.OS !== 'web';

export default function PlaylistDetailScreen() {
  const route = useRoute<Route>();
  const nav = useNavigation<any>();
  const { playlistId } = route.params;

  const [playlist, setPlaylist] = useState<Playlist | null>(null);
  const [tracks, setTracks] = useState<PlaylistTrack[]>([]);
  const [version, setVersion] = useState(0);
  const [showSearch, setShowSearch] = useState(false);
  const [searchQ, setSearchQ] = useState('');
  const [searchResults, setSearchResults] = useState<Track[]>([]);
  const [error, setError] = useState('');
  const [loadFailed, setLoadFailed] = useState(false);
  const [searchError, setSearchError] = useState('');
  const versionRef = useRef(0);
  const { user } = useAuth();
  const isOwner = !!playlist && !!user && playlist.owner_id === user.id;
  const tracksRef = useRef<PlaylistTrack[]>([]);
  tracksRef.current = tracks;
  const player = usePreviewPlayer((finishedId) => {
    const list = tracksRef.current;
    const next = list[list.findIndex((t) => t.id === finishedId) + 1];
    if (next) player.play(next);
  });

  useFocusEffect(useCallback(() => { load(); }, [playlistId]));

  useWebSocket('playlists', playlistId, (msg) => {
    if (msg.type === 'playlist:updated') load();
  });

  async function load() {
    setError('');
    try {
      const [pl, ts] = await Promise.all([
        playlistsApi.get(playlistId),
        playlistsApi.tracks(playlistId),
      ]);
      setPlaylist(pl);
      setTracks(ts);
      setVersion(pl.version);
      versionRef.current = pl.version;
      setLoadFailed(false);
    } catch (e) {
      setLoadFailed(true);
      setError(apiError(e, 'COULD NOT LOAD THIS PLAYLIST'));
    }
  }

  async function moveTrack(trackId: string, afterId: string | null) {
    setError('');
    for (let attempt = 0; ; attempt++) {
      try {
        const res = await playlistsApi.moveTrack(playlistId, trackId, {
          after_track_id: afterId,
          version: versionRef.current,
        });
        setVersion(res.version);
        versionRef.current = res.version;
        if (attempt > 0) await load();
        return;
      } catch (e: any) {
        if (e?.response?.status !== 409 || attempt === MOVE_MAX_RETRIES) {
          setError(apiError(e, 'COULD NOT MOVE THIS TRACK'));
          await load();
          return;
        }
        await backoff(attempt);
        await load();
      }
    }
  }

  async function onDragEnd({ data, from, to }: { data: PlaylistTrack[]; from: number; to: number }) {
    if (from === to) return;
    setTracks(data);
    const moved = data[to];
    const afterId = to > 0 ? data[to - 1].id : null;
    await moveTrack(moved.id, afterId);
  }

  async function reorder(from: number, to: number) {
    if (from === to || to < 0 || to >= tracks.length) return;
    const data = [...tracks];
    const [moved] = data.splice(from, 1);
    data.splice(to, 0, moved);
    setTracks(data);
    const afterId = to > 0 ? data[to - 1].id : null;
    await moveTrack(moved.id, afterId);
  }

  async function deleteTrack(trackId: string) {
    setError('');
    try {
      await playlistsApi.deleteTrack(playlistId, trackId);
    } catch (e) {
      setError(apiError(e, 'COULD NOT REMOVE THIS TRACK'));
      return;
    }
    load();
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

  async function addTrack(track: Track) {
    setSearchError('');
    try {
      await playlistsApi.addTrack(playlistId, track.deezer_id);
    } catch (e) {
      setSearchError(apiError(e, 'COULD NOT ADD THIS TRACK'));
      return;
    }
    setShowSearch(false);
    setSearchResults([]);
    setSearchQ('');
    load();
  }

  function renderItem({ item, drag, isActive, getIndex }: RenderItemParams<PlaylistTrack>) {
    const index = getIndex() ?? 0;
    return (
      <ScaleDecorator>
        <Pressable
          onLongPress={CAN_DRAG ? drag : undefined}
          style={[styles.track, isActive && styles.trackActive]}
        >
          {CAN_DRAG ? (
            <Text size="xs" dim style={styles.handle}>{'≡'}</Text>
          ) : (
            <View style={styles.moveBtns}>
              <Pressable
                onPress={() => reorder(index, index - 1)}
                disabled={index === 0}
                style={styles.moveBtn}
                accessibilityLabel={`Move ${item.title} up`}
              >
                <Text size="xs" dim={index === 0} accent={index > 0}>{'▲'}</Text>
              </Pressable>
              <Pressable
                onPress={() => reorder(index, index + 1)}
                disabled={index === tracks.length - 1}
                style={styles.moveBtn}
                accessibilityLabel={`Move ${item.title} down`}
              >
                <Text size="xs" dim={index === tracks.length - 1} accent={index < tracks.length - 1}>{'▼'}</Text>
              </Pressable>
            </View>
          )}
          <View style={{ flex: 1 }}>
            <Text size="sm" accent={player.playingId === item.id}>{item.title}</Text>
            <Text size="xs" dim>{item.artist_name}</Text>
          </View>
          <Pressable
            onPress={() => player.toggle(item)}
            style={styles.previewBtn}
            accessibilityRole="button"
            accessibilityLabel={`Preview ${item.title}`}
          >
            <Text size="xs" accent>
              {player.loadingId === item.id ? '...' : player.playingId === item.id ? '■ STOP' : '▶ PREVIEW'}
            </Text>
          </Pressable>
          <Pressable onPress={() => deleteTrack(item.id)} style={styles.deleteBtn}>
            <Text size="sm" danger>{'✕'}</Text>
          </Pressable>
        </Pressable>
      </ScaleDecorator>
    );
  }

  if (!playlist && loadFailed) {
    return (
      <Screen title="PLAYLIST">
        <Text size="sm" danger>{`! ${error}`}</Text>
        <View style={{ height: spacing.md }} />
        <Button label="RETRY" onPress={load} />
      </Screen>
    );
  }
  if (!playlist) return <Screen title="LOADING..."><Text dim>{'// FETCHING...'}</Text></Screen>;

  return (
    <Screen title={playlist.name} padded={false}>
      <View style={styles.header}>
        <Text size="xs" dim>{`// v${version} — ${playlist.visibility.toUpperCase()} — ${playlist.license.toUpperCase()}`}</Text>
        <PlaylistByline ownerId={playlist.owner_id} ownerName={playlist.owner_display_name} />
        {playlist.description ? <Text size="xs" dim>{playlist.description}</Text> : null}
        {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
        {player.error ? <Text size="sm" danger>{`! ${player.error}`}</Text> : null}
        {tracks.length > 0 && (
          <Button
            label={player.playingId || player.loadingId ? '■ STOP PREVIEWS' : '▶ PREVIEW THE PLAYLIST'}
            onPress={() => (player.playingId || player.loadingId ? player.stop() : player.play(tracks[0]))}
            variant="ghost"
          />
        )}
        {tracks.length > 0 && <Text size="xs" dim>{'// DEEZER ONLY SHARES 30-SECOND PREVIEWS; EACH ONE PLAYS TO THE END, THEN THE NEXT TRACK STARTS'}</Text>}
        <Button label="+ ADD TRACK" onPress={() => { setSearchError(''); setShowSearch(true); }} />
        {isOwner && (
          <Button label="EDIT PLAYLIST" onPress={() => nav.navigate('EditPlaylist', { playlistId })} variant="ghost" />
        )}
      </View>

      <DraggableFlatList
        data={tracks}
        keyExtractor={(t) => t.id}
        renderItem={renderItem}
        onDragEnd={onDragEnd}
        contentContainerStyle={styles.list}
        ListEmptyComponent={<Text dim style={styles.empty}>{'// NO TRACKS YET. ADD ONE TO GET STARTED.'}</Text>}
      />

      <Modal visible={showSearch} animationType="slide" onRequestClose={() => setShowSearch(false)}>
        <View style={styles.modal}>
          <Text size="md" accent>{'■ ADD TRACK'}</Text>
          {searchError ? <Text size="sm" danger>{`! ${searchError}`}</Text> : null}
          <View style={styles.searchRow}>
            <TextInput
              style={styles.searchInput}
              placeholder="SEARCH..."
              placeholderTextColor={colors.textDim}
              value={searchQ}
              onChangeText={setSearchQ}
              onSubmitEditing={search}
              returnKeyType="search"
            />
            <Pressable onPress={search} style={styles.searchBtn}>
              <Text accent>{'GO'}</Text>
            </Pressable>
          </View>
          <FlatList
            data={searchResults}
            keyExtractor={(t) => String(t.deezer_id)}
            renderItem={({ item }) => (
              <Pressable style={styles.result} onPress={() => addTrack(item)}>
                <Text size="sm" accent>{'+ '}</Text>
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
  header: { padding: spacing.md, borderBottomWidth: 1, borderBottomColor: colors.border },
  list: { padding: spacing.md },
  empty: { marginTop: spacing.lg },
  track: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  trackActive: { backgroundColor: colors.accentGlow },
  handle: { marginRight: spacing.sm, fontSize: 22 },
  moveBtns: { marginRight: spacing.sm, justifyContent: 'center' },
  moveBtn: { paddingHorizontal: spacing.xs, paddingVertical: 1 },
  deleteBtn: { padding: spacing.xs },
  previewBtn: { paddingHorizontal: spacing.sm, paddingVertical: spacing.xs },
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
