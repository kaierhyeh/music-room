import React, { useState, useEffect } from 'react';
import { View } from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import * as Linking from 'expo-linking';
import { authApi } from '../../api/auth';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';
import { spacing } from '../../theme';
import { AuthStackParams } from '../../navigation/AuthNavigator';
import { apiError } from '../../api/error';

type Nav = StackNavigationProp<AuthStackParams, 'VerifyEmail'>;
type Route = RouteProp<AuthStackParams, 'VerifyEmail'>;

export default function VerifyEmailScreen() {
  const nav = useNavigation<Nav>();
  const route = useRoute<Route>();
  const [token, setToken] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const sub = Linking.addEventListener('url', ({ url }) => {
      const parsed = Linking.parse(url);
      const t = parsed.queryParams?.token;
      if (t && typeof t === 'string') verify(t);
    });
    return () => sub.remove();
  }, []);

  async function verify(t: string) {
    setError('');
    setLoading(true);
    try {
      await authApi.verifyEmail(t);
      nav.navigate('Login');
    } catch (e: any) {
      setError(apiError(e, 'THIS CODE IS NOT VALID.'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Screen title="VERIFY EMAIL" scroll>
      <Text size="sm" dim>{`// CHECK EMAIL: ${route.params?.email ?? '?'}`}</Text>
      <Text size="sm" dim>{'// OR PASTE TOKEN BELOW'}</Text>
      <View style={{ height: spacing.md }} />
      <Input label="VERIFICATION TOKEN" value={token} onChangeText={setToken} autoCapitalize="none" />
      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
      <View style={{ height: spacing.md }} />
      <Button label={loading ? 'VERIFYING...' : 'VERIFY'} onPress={() => verify(token)} disabled={loading} />
    </Screen>
  );
}
