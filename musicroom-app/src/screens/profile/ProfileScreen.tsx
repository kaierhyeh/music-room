import React, { useEffect, useState } from 'react';
import { View, ScrollView, StyleSheet } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { useAuth } from '../../context/AuthContext';
import { subscriptionApi, Subscription } from '../../api/subscription';
import { apiError } from '../../api/error';
import { setBaseUrl, getBaseUrl } from '../../storage/storage';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Button from '../../components/ui/Button';
import Input from '../../components/ui/Input';
import MusicPreferencesBlock from '../../components/MusicPreferences';
import { colors, spacing } from '../../theme';

export default function ProfileScreen() {
  const nav = useNavigation<any>();
  const { user, logout, refreshUser, linkGoogle } = useAuth();
  const [editUrl, setEditUrl] = useState(false);
  const [url, setUrl] = useState('');
  const [linking, setLinking] = useState(false);
  const [linkError, setLinkError] = useState('');
  const [sub, setSub] = useState<Subscription | null>(null);
  const [switching, setSwitching] = useState(false);
  const [subError, setSubError] = useState('');

  useEffect(() => { loadSubscription(); }, []);

  async function loadSubscription() {
    setSubError('');
    try {
      setSub(await subscriptionApi.get());
    } catch (e) {
      setSub(null);
      setSubError(apiError(e, 'COULD NOT READ THE SUBSCRIPTION'));
    }
  }

  async function switchTier(tier: 'free' | 'premium') {
    setSwitching(true);
    setSubError('');
    try {
      setSub(await subscriptionApi.set(tier));
      await refreshUser();
    } catch (e: any) {
      setSubError(apiError(e, 'COULD NOT CHANGE THE SUBSCRIPTION'));
    } finally {
      setSwitching(false);
    }
  }

  async function openUrlEdit() {
    const current = await getBaseUrl();
    setUrl(current);
    setEditUrl(true);
  }

  async function saveUrl() {
    await setBaseUrl(url.trim());
    setEditUrl(false);
    loadSubscription();
  }

  async function onLinkGoogle() {
    setLinking(true);
    setLinkError('');
    try {
      await linkGoogle();
    } catch (e: any) {
      setLinkError((e?.message ?? 'could not link google').toUpperCase());
    } finally {
      setLinking(false);
    }
  }

  if (!user) return null;

  return (
    <Screen title="PROFILE" scroll>
      <View style={styles.field}>
        <Text size="sm" dim>{'// ID'}</Text>
        <Text size="sm" style={styles.mono}>{user.id}</Text>
      </View>
      <View style={styles.field}>
        <Text size="sm" dim>{'// DISPLAY NAME'}</Text>
        <Text size="md" accent>{user.display_name}</Text>
      </View>
      {user.email && (
        <View style={styles.field}>
          <Text size="sm" dim>{'// EMAIL'}</Text>
          <Text size="sm">{user.email}</Text>
        </View>
      )}

      <MusicPreferencesBlock prefs={user.music_preferences} />

      <View style={styles.divider} />

      <View style={styles.field}>
        <Text size="sm" dim>{'// SUBSCRIPTION'}</Text>
        <Text size="md" accent>{(sub?.tier ?? '...').toUpperCase()}</Text>
        {sub?.expires_at ? (
          <Text size="xs" dim>{`// RENEWS ${new Date(sub.expires_at).toISOString().slice(0, 10)}`}</Text>
        ) : null}
        <Text size="xs" dim>
          {sub?.tier === 'premium'
            ? '// COLLABORATIVE PLAYLISTS UNLOCKED'
            : '// PREMIUM UNLOCKS COLLABORATIVE PLAYLISTS'}
        </Text>
      </View>
      {!sub && subError ? (
        <Button label="RETRY" onPress={loadSubscription} variant="ghost" />
      ) : sub?.tier === 'premium' ? (
        <Button
          label={switching ? 'SWITCHING...' : 'CANCEL PREMIUM'}
          onPress={() => switchTier('free')}
          disabled={switching}
          variant="ghost"
        />
      ) : (
        <Button
          label={switching ? 'SWITCHING...' : 'UPGRADE TO PREMIUM'}
          onPress={() => switchTier('premium')}
          disabled={switching}
        />
      )}
      {subError ? <Text size="sm" danger>{`! ${subError}`}</Text> : null}

      <View style={styles.divider} />

      <Button label="EDIT PROFILE" onPress={() => nav.navigate('EditProfile')} />
      {!user.google_linked && (
        <Button
          label={linking ? 'OPENING GOOGLE...' : 'LINK GOOGLE ACCOUNT'}
          onPress={onLinkGoogle}
          disabled={linking}
          variant="ghost"
        />
      )}
      {linkError ? <Text size="sm" danger>{`! ${linkError}`}</Text> : null}

      <View style={styles.divider} />

      {editUrl ? (
        <>
          <Input label="BACKEND URL" value={url} onChangeText={setUrl} autoCapitalize="none" />
          <Button label="SAVE" onPress={saveUrl} />
          <Button label="CANCEL" onPress={() => setEditUrl(false)} variant="ghost" />
        </>
      ) : (
        <Button label="CONFIGURE SERVER URL" onPress={openUrlEdit} variant="ghost" />
      )}

      <View style={styles.divider} />
      <Button label="LOGOUT" onPress={logout} variant="danger" />
    </Screen>
  );
}

const styles = StyleSheet.create({
  field: { marginVertical: spacing.xs },
  mono: { letterSpacing: 0.5 },
  divider: { height: 1, backgroundColor: colors.border, marginVertical: spacing.md },
});
