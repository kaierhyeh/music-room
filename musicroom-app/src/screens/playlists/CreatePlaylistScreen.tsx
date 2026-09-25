import React, { useEffect, useState } from 'react';
import { View, StyleSheet, Pressable } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { playlistsApi } from '../../api/playlists';
import { friendsApi } from '../../api/friends';
import { User } from '../../api/users';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';
import { colors, spacing } from '../../theme';
import { apiError } from '../../api/error';

export default function CreatePlaylistScreen() {
  const nav = useNavigation<any>();
  const [name, setName] = useState('');
  const [desc, setDesc] = useState('');
  const [visibility, setVisibility] = useState<'public' | 'private'>('public');
  const [license, setLicense] = useState<'open' | 'invited_only'>('open');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const [friends, setFriends] = useState<User[]>([]);
  const [friendsError, setFriendsError] = useState('');
  const [invites, setInvites] = useState<Record<string, boolean>>({});

  useEffect(() => {
    friendsApi.list().then(setFriends, (e) => setFriendsError(apiError(e, 'COULD NOT LOAD YOUR FRIENDS')));
  }, []);

  function pick(id: string, canEdit: boolean) {
    setInvites((prev) => {
      const next = { ...prev };
      if (next[id] === canEdit) delete next[id];
      else next[id] = canEdit;
      return next;
    });
  }

  async function create() {
    setError('');
    setLoading(true);
    try {
      const collaborators = Object.entries(invites).map(([user_id, can_edit]) => ({ user_id, can_edit }));
      const pl = await playlistsApi.create({ name, description: desc, visibility, license, collaborators });
      nav.replace('PlaylistDetail', { playlistId: pl.id });
    } catch (e: any) {
      setError(apiError(e, 'COULD NOT CREATE IT. TRY AGAIN.'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Screen title="NEW PLAYLIST" scroll>
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

      <View style={styles.divider} />
      <Text size="sm" dim>{'// INVITE FRIENDS'}</Text>
      <Text size="xs" dim style={styles.hint}>
        {'// VIEW — CAN OPEN IT, EVEN PRIVATE, NOT CHANGE IT\n// EDIT (PREMIUM) — CAN ALSO CHANGE THE TRACKS'}
      </Text>
      {friendsError ? (
        <Text size="sm" danger>{`! ${friendsError}`}</Text>
      ) : friends.length === 0 ? (
        <Text dim size="xs" style={styles.empty}>{'// NO FRIENDS TO INVITE'}</Text>
      ) : (
        friends.map((f) => {
          const right = invites[f.id];
          return (
            <View key={f.id} style={styles.friendRow}>
              <Text size="sm" style={styles.friendName} dim={right === undefined}>{f.display_name}</Text>
              <Pressable
                onPress={() => pick(f.id, false)}
                style={styles.roleBtn}
                accessibilityRole="checkbox"
                accessibilityState={{ checked: right === false }}
                accessibilityLabel={`Invite ${f.display_name} to view`}
              >
                <Text size="xs" accent={right === false} dim={right !== false}>{right === false ? '[VIEW]' : ' VIEW '}</Text>
              </Pressable>
              <Pressable
                onPress={() => pick(f.id, true)}
                style={styles.roleBtn}
                accessibilityRole="checkbox"
                accessibilityState={{ checked: right === true }}
                accessibilityLabel={`Invite ${f.display_name} to edit`}
              >
                <Text size="xs" accent={right === true} dim={right !== true}>{right === true ? '[EDIT]' : ' EDIT '}</Text>
              </Pressable>
            </View>
          );
        })
      )}

      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
      <View style={{ height: spacing.md }} />
      <Button label={loading ? 'CREATING...' : 'CREATE'} onPress={create} disabled={loading} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  divider: { height: 1, backgroundColor: colors.border, marginVertical: spacing.sm },
  row: { flexDirection: 'row', gap: 8 },
  hint: { marginBottom: spacing.xs },
  empty: { marginVertical: spacing.xs },
  friendRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: spacing.xs,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  friendName: { flex: 1 },
  roleBtn: { paddingVertical: spacing.xs, paddingHorizontal: spacing.sm },
});
