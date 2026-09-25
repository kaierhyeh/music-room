import React, { useEffect, useState } from 'react';
import { View, StyleSheet } from 'react-native';
import { useRoute, RouteProp } from '@react-navigation/native';
import { usersApi, User } from '../../api/users';
import { friendsApi } from '../../api/friends';
import { apiError } from '../../api/error';
import { useAuth } from '../../context/AuthContext';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Button from '../../components/ui/Button';
import MusicPreferencesBlock from '../../components/MusicPreferences';
import { colors, spacing } from '../../theme';

export default function UserProfileScreen() {
  const route = useRoute<RouteProp<{ UserProfile: { userId: string } }, 'UserProfile'>>();
  const { user: me } = useAuth();
  const isSelf = me?.id === route.params.userId;
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState('');
  const [requestError, setRequestError] = useState('');
  const [requested, setRequested] = useState(false);
  const [isFriend, setIsFriend] = useState(false);

  useEffect(() => { load(); }, [route.params.userId]);

  async function load() {
    setError('');
    try {
      setUser(await usersApi.getById(route.params.userId));
    } catch (e) {
      setError(apiError(e, 'USER NOT FOUND'));
      return;
    }
    friendsApi.list()
      .then((friends) => setIsFriend(friends.some((f) => f.id === route.params.userId)))
      .catch(() => {});
  }

  async function addFriend() {
    setRequestError('');
    try {
      await friendsApi.send(route.params.userId);
      setRequested(true);
    } catch (e) {
      setRequestError(apiError(e, 'COULD NOT SEND THE REQUEST'));
    }
  }

  if (error) {
    return (
      <Screen title="USER">
        <Text danger>{`! ${error}`}</Text>
        <View style={{ height: spacing.md }} />
        <Button label="RETRY" onPress={load} />
      </Screen>
    );
  }
  if (!user) return <Screen title="LOADING..."><Text dim>{'// FETCHING DATA...'}</Text></Screen>;

  return (
    <Screen title="USER PROFILE" scroll>
      <Text size="lg" accent>{user.display_name}</Text>
      {user.profile_data?.bio && (
        <View style={styles.field}>
          <Text size="sm" dim>{'// BIO'}</Text>
          <Text size="sm">{user.profile_data.bio}</Text>
        </View>
      )}
      {user.profile_data?.location && (
        <View style={styles.field}>
          <Text size="sm" dim>{'// LOCATION'}</Text>
          <Text size="sm">{user.profile_data.location}</Text>
        </View>
      )}
      {user.music_preferences && <MusicPreferencesBlock prefs={user.music_preferences} />}
      <View style={{ height: spacing.md }} />
      {requestError ? <Text size="sm" danger>{`! ${requestError}`}</Text> : null}
      {isSelf ? (
        <Text size="sm" dim>{'// THIS IS YOU'}</Text>
      ) : isFriend ? (
        <Text size="sm" dim>{'// ALREADY FRIENDS'}</Text>
      ) : requested ? (
        <Text size="sm" dim>{'// REQUEST SENT'}</Text>
      ) : (
        <Button label="SEND FRIEND REQUEST" onPress={addFriend} />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  field: { marginVertical: spacing.xs },
});
