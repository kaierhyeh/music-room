import React, { useState } from 'react';
import { View } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { usersApi, User } from '../../api/users';
import { friendsApi } from '../../api/friends';
import { apiError } from '../../api/error';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';
import { spacing } from '../../theme';

export default function UserSearchScreen() {
  const nav = useNavigation<any>();
  const [uuid, setUuid] = useState('');
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [requested, setRequested] = useState(false);
  const [isFriend, setIsFriend] = useState(false);

  async function lookup() {
    setError('');
    setUser(null);
    setRequested(false);
    setIsFriend(false);
    setLoading(true);
    try {
      const [found, friends] = await Promise.all([
        usersApi.getById(uuid.trim()),
        friendsApi.list().catch(() => []),
      ]);
      setIsFriend(friends.some((f) => f.id === found.id));
      setUser(found);
    } catch (e) {
      setError(apiError(e, 'USER NOT FOUND'));
    } finally {
      setLoading(false);
    }
  }

  async function sendRequest() {
    if (!user) return;
    setError('');
    try {
      await friendsApi.send(user.id);
      setRequested(true);
    } catch (e) {
      setError(apiError(e, 'COULD NOT SEND THE REQUEST'));
    }
  }

  return (
    <Screen title="FIND USER" scroll>
      <Text size="sm" dim>{'// PASTE USER UUID'}</Text>
      <Input label="UUID" value={uuid} onChangeText={setUuid} autoCapitalize="none" />
      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
      <View style={{ height: spacing.sm }} />
      <Button label={loading ? 'SEARCHING...' : 'LOOKUP'} onPress={lookup} disabled={loading} />
      {user && (
        <>
          <View style={{ height: spacing.md }} />
          <Text size="md" accent>{`> ${user.display_name}`}</Text>
          {isFriend ? (
            <Text size="sm" dim>{'// ALREADY FRIENDS'}</Text>
          ) : requested ? (
            <Text size="sm" dim>{'// REQUEST SENT'}</Text>
          ) : (
            <Button label="SEND FRIEND REQUEST" onPress={sendRequest} />
          )}
          <Button label="VIEW PROFILE" onPress={() => nav.navigate('UserProfile', { userId: user.id })} variant="ghost" />
        </>
      )}
    </Screen>
  );
}
