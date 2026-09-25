import React, { useState, useEffect } from 'react';
import { View, StyleSheet, Image } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { useAuth } from '../../context/AuthContext';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';
import { colors, spacing } from '../../theme';
import { AuthStackParams } from '../../navigation/AuthNavigator';
import { getBaseUrl, setBaseUrl } from '../../storage/storage';
import { apiError } from '../../api/error';

type Nav = StackNavigationProp<AuthStackParams, 'Login'>;

export default function LoginScreen() {
  const nav = useNavigation<Nav>();
  const { login, loginWithGoogle } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [editUrl, setEditUrl] = useState(false);
  const [googleLoading, setGoogleLoading] = useState(false);
  const [serverUrl, setServerUrl] = useState('');

  useEffect(() => {
    getBaseUrl().then(setServerUrl);
  }, []);

  async function saveUrl() {
    await setBaseUrl(serverUrl.trim());
    setEditUrl(false);
  }

  async function submit() {
    setError('');
    setLoading(true);
    try {
      await login(email, password);
    } catch (e: any) {
      setError(apiError(e, 'LOGIN FAILED'));
    } finally {
      setLoading(false);
    }
  }

  async function submitGoogle() {
    setError('');
    setGoogleLoading(true);
    try {
      await loginWithGoogle();
    } catch (e: any) {
      setError(apiError(e, (e?.message ?? 'GOOGLE SIGN-IN FAILED').toUpperCase()));
    } finally {
      setGoogleLoading(false);
    }
  }

  return (
    <Screen scroll>
      <View style={styles.hero}>
        <Text size="xxl" accent upper style={styles.glitch}>MUSIC</Text>
        <Text size="xxl" accent upper style={[styles.glitch, styles.glitch2]}>ROOM</Text>
        <Text size="sm" dim style={styles.sub}>{'> SYSTEM ONLINE'}</Text>
      </View>

      <View style={styles.divider} />

      <Input label="EMAIL" value={email} onChangeText={setEmail} keyboardType="email-address" autoCapitalize="none" />
      <Input label="PASSWORD" value={password} onChangeText={setPassword} secureTextEntry />

      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}

      <View style={styles.gap} />
      <Button label={loading ? 'CONNECTING...' : 'ENTER'} onPress={submit} disabled={loading} />
      <Button
        label={googleLoading ? 'OPENING GOOGLE...' : 'SIGN IN WITH GOOGLE'}
        onPress={submitGoogle}
        disabled={googleLoading}
        variant="ghost"
      />
      <Button label="REGISTER NEW ACCOUNT" onPress={() => nav.navigate('Register')} variant="ghost" />
      <Button label="FORGOT PASSWORD" onPress={() => nav.navigate('ForgotPassword')} variant="ghost" />

      <View style={styles.divider} />

      {editUrl ? (
        <>
          <Input label="BACKEND URL" value={serverUrl} onChangeText={setServerUrl} autoCapitalize="none" />
          <Button label="SAVE" onPress={saveUrl} />
          <Button label="CANCEL" onPress={() => setEditUrl(false)} variant="ghost" />
        </>
      ) : (
        <Button label="CONFIGURE SERVER URL" onPress={() => setEditUrl(true)} variant="ghost" />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: {
    alignItems: 'center',
    paddingVertical: spacing.xxl,
  },
  glitch: {
    textShadowColor: colors.danger,
    textShadowOffset: { width: 2, height: 0 },
    textShadowRadius: 0,
    letterSpacing: 8,
  },
  glitch2: {
    textShadowColor: colors.accentDim,
    textShadowOffset: { width: -2, height: 0 },
    marginTop: -8,
  },
  sub: {
    marginTop: spacing.md,
    letterSpacing: 3,
  },
  divider: {
    height: 1,
    backgroundColor: colors.accentDim,
    marginVertical: spacing.lg,
  },
  gap: { height: spacing.md },
});
