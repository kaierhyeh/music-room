import React, { useCallback, useEffect, useState } from 'react';
import { View, StyleSheet, Pressable } from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { playlistsApi, Playlist, PlaylistLicense, PlaylistVisibility, Collaborator } from '../../api/playlists';
import { friendsApi } from '../../api/friends';
import { User } from '../../api/users';
import { apiError } from '../../api/error';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';
import { colors, spacing } from '../../theme';

type Route = RouteProp<{ EditPlaylist: { playlistId: string } }, 'EditPlaylist'>;

export default function EditPlaylistScreen() {
  const nav = useNavigation<any>();
  const { playlistId } = useRoute<Route>().params;

  const [playlist, setPlaylist] = useState<Playlist | null>(null);
  const [name, setName] = useState('');
  const [desc, setDesc] = useState('');
  const [visibility, setVisibility] = useState<PlaylistVisibility>('public');
  const [license, setLicense] = useState<PlaylistLicense>('open');

  const [friends, setFriends] = useState<User[]>([]);
  const [collaborators, setCollaborators] = useState<Collaborator[]>([]);

  const [loadError, setLoadError] = useState('');
  const [error, setError] = useState('');
  const [collabError, setCollabError] = useState('');
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoadError('');
    try {
      const pl = await playlistsApi.get(playlistId);
      setPlaylist(pl);
      setName(pl.name);
      setDesc(pl.description ?? '');
      setVisibility(pl.visibility);
      setLicense(pl.license);
    } catch (e) {
      setLoadError(apiError(e, 'COULD NOT LOAD THIS PLAYLIST'));
      return;
    }
    setCollabError('');
    try {
      const [fr, co] = await Promise.all([friendsApi.list(), playlistsApi.collaborators(playlistId)]);
      setFriends(fr);
      setCollaborators(co);
    } catch (e) {
      setCollabError(apiError(e, 'COULD NOT LOAD COLLABORATORS'));
    }
  }, [playlistId]);

  useEffect(() => { load(); }, [load]);

  async function save() {
    if (!name.trim()) {
      setError('NAME CANNOT BE EMPTY');
      return;
    }
    setError('');
    setSaving(true);
    try {
      await playlistsApi.update(playlistId, {
        name: name.trim(),
        description: desc.trim(),
        visibility,
        license,
      });
      nav.goBack();
    } catch (e) {
      setError(apiError(e, 'COULD NOT SAVE THIS PLAYLIST'));
    } finally {
      setSaving(false);
    }
  }

  async function addCollaborator(user: { id: string }, canEdit: boolean) {
    setCollabError('');
    try {
      await playlistsApi.addCollaborator(playlistId, user.id, canEdit);
      setCollaborators(await playlistsApi.collaborators(playlistId));
    } catch (e) {
      setCollabError(apiError(e, 'COULD NOT ADD COLLABORATOR'));
    }
  }

  async function removeCollaborator(c: Collaborator) {
    setCollabError('');
    try {
      await playlistsApi.removeCollaborator(playlistId, c.id);
      setCollaborators(await playlistsApi.collaborators(playlistId));
    } catch (e) {
      setCollabError(apiError(e, 'COULD NOT REMOVE COLLABORATOR'));
    }
  }

  if (loadError) {
    return (
      <Screen title="EDIT PLAYLIST">
        <Text danger>{`! ${loadError}`}</Text>
        <View style={{ height: spacing.md }} />
        <Button label="RETRY" onPress={load} />
      </Screen>
    );
  }
  if (!playlist) return <Screen title="LOADING..."><Text dim>{'// FETCHING...'}</Text></Screen>;

  return (
    <Screen title="EDIT PLAYLIST" scroll>
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
          ? '// ANYONE CAN FIND AND OPEN THIS PLAYLIST'
          : '// ONLY YOU AND THE FRIENDS YOU INVITE CAN OPEN IT'}
      </Text>

      <View style={styles.divider} />
      <Text size="sm" dim>{'// LICENSE'}</Text>
      <View style={styles.row}>
        <Button label="OPEN" onPress={() => setLicense('open')} variant={license === 'open' ? 'primary' : 'ghost'} />
        <Button label="INVITE ONLY" onPress={() => setLicense('invited_only')} variant={license === 'invited_only' ? 'primary' : 'ghost'} />
      </View>
      <Text size="xs" dim>
        {license === 'open'
          ? '// ANYONE WHO CAN SEE IT CAN EDIT THE TRACKS, EXCEPT VIEWERS'
          : '// ONLY YOU AND YOUR EDITORS CAN EDIT THE TRACKS'}
      </Text>

      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
      <View style={{ height: spacing.sm }} />
      <Button label={saving ? 'SAVING...' : 'SAVE'} onPress={save} disabled={saving} />

      <View style={styles.divider} />
      <Text size="sm" dim>{'// INVITED FRIENDS'}</Text>
      <Text size="xs" dim style={styles.hint}>
        {'// VIEW — CAN OPEN IT, EVEN PRIVATE, NOT CHANGE IT\n// EDIT (PREMIUM) — CAN ALSO CHANGE THE TRACKS'}
      </Text>
      {collabError ? <Text size="sm" danger>{`! ${collabError}`}</Text> : null}

      {collaborators.length === 0 ? (
        <Text dim size="xs" style={styles.empty}>{'// NOBODY INVITED YET'}</Text>
      ) : (
        collaborators.map((c) => (
          <View key={c.id} style={styles.collabRow}>
            <Pressable
              style={styles.collabName}
              onPress={() => nav.navigate('UserProfile', { userId: c.id })}
              accessibilityRole="link"
              accessibilityLabel={`Open ${c.display_name}'s profile`}
            >
              <Text size="sm" accent>{`> ${c.display_name}`}</Text>
            </Pressable>
            <Pressable
              onPress={() => addCollaborator(c, !c.can_edit)}
              style={styles.roleBtn}
              accessibilityLabel={`${c.display_name} can ${c.can_edit ? 'edit' : 'view'} — switch`}
            >
              <Text size="xs" accent={c.can_edit}>{c.can_edit ? '[EDIT]' : '[VIEW]'}</Text>
            </Pressable>
            <Pressable
              onPress={() => removeCollaborator(c)}
              style={styles.removeBtn}
              accessibilityLabel={`Remove ${c.display_name}`}
            >
              <Text size="sm" danger>{'✕'}</Text>
            </Pressable>
          </View>
        ))
      )}

      <View style={{ height: spacing.sm }} />
      <Text size="xs" dim>{'// ADD A FRIEND'}</Text>
      {friends.length === 0 ? (
        <Text dim size="xs" style={styles.empty}>{'// NO FRIENDS TO INVITE'}</Text>
      ) : (
        friends.map((f) => {
          const already = collaborators.some((c) => c.id === f.id);
          return (
            <View key={f.id} style={styles.friendRow}>
              <Text size="sm" accent>{already ? '✓ ' : '  '}</Text>
              <Text size="sm" dim={already} style={styles.friendName}>{f.display_name}</Text>
              {!already && (
                <>
                  <Pressable
                    onPress={() => addCollaborator(f, false)}
                    style={styles.roleBtn}
                    accessibilityLabel={`Invite ${f.display_name} to view`}
                  >
                    <Text size="xs" accent>{'+ VIEW'}</Text>
                  </Pressable>
                  <Pressable
                    onPress={() => addCollaborator(f, true)}
                    style={styles.roleBtn}
                    accessibilityLabel={`Invite ${f.display_name} to edit`}
                  >
                    <Text size="xs" accent>{'+ EDIT'}</Text>
                  </Pressable>
                </>
              )}
            </View>
          );
        })
      )}
      <View style={{ height: spacing.lg }} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  divider: { height: 1, backgroundColor: colors.border, marginVertical: spacing.sm },
  row: { flexDirection: 'row', gap: 8 },
  hint: { marginBottom: spacing.xs },
  empty: { marginVertical: spacing.xs },
  collabRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: spacing.xs,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  collabName: { flex: 1, paddingVertical: spacing.xs },
  removeBtn: { padding: spacing.xs },
  roleBtn: { paddingVertical: spacing.xs, paddingHorizontal: spacing.sm },
  friendName: { flex: 1 },
  friendRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
});
