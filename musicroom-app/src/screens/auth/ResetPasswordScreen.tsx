import React, { useState } from 'react';
import { View } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { authApi } from '../../api/auth';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';
import { spacing } from '../../theme';
import { AuthStackParams } from '../../navigation/AuthNavigator';
import { apiError } from '../../api/error';

type Nav = StackNavigationProp<AuthStackParams, 'ResetPassword'>;

export default function ResetPasswordScreen() {
  const nav = useNavigation<Nav>();
  const [token, setToken] = useState('');
  const [password, setPassword] = useState('');
  const [done, setDone] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function submit() {
    setError('');
    setLoading(true);
    try {
      await authApi.resetPassword(token, password);
      setDone(true);
    } catch (e: any) {
      setError(apiError(e, 'SOMETHING WENT WRONG. TRY AGAIN.'));
    } finally {
      setLoading(false);
    }
  }

  if (done) {
    return (
      <Screen title="PASSWORD RESET">
        <Text size="sm" accent>{'> PASSWORD UPDATED'}</Text>
        <View style={{ height: spacing.md }} />
        <Button label="BACK TO LOGIN" onPress={() => nav.navigate('Login')} />
      </Screen>
    );
  }

  return (
    <Screen title="RESET PASSWORD" scroll>
      <Input label="RESET TOKEN" value={token} onChangeText={setToken} autoCapitalize="none" />
      <Input label="NEW PASSWORD" value={password} onChangeText={setPassword} secureTextEntry />
      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
      <View style={{ height: spacing.md }} />
      <Button label={loading ? 'RESETTING...' : 'RESET PASSWORD'} onPress={submit} disabled={loading} />
    </Screen>
  );
}
