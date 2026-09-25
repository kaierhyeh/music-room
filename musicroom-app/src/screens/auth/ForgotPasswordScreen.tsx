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

type Nav = StackNavigationProp<AuthStackParams, 'ForgotPassword'>;

export default function ForgotPasswordScreen() {
  const nav = useNavigation<Nav>();
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function submit() {
    setError('');
    setLoading(true);
    try {
      await authApi.forgotPassword(email);
      setSent(true);
    } catch (e: any) {
      setError(apiError(e, 'SOMETHING WENT WRONG. TRY AGAIN.'));
    } finally {
      setLoading(false);
    }
  }

  if (sent) {
    return (
      <Screen title="CHECK EMAIL">
        <Text size="sm" dim>{'// RESET LINK SENT'}</Text>
        <View style={{ height: spacing.md }} />
        <Button label="ENTER TOKEN" onPress={() => nav.navigate('ResetPassword')} />
      </Screen>
    );
  }

  return (
    <Screen title="FORGOT PASSWORD" scroll>
      <Input label="EMAIL" value={email} onChangeText={setEmail} keyboardType="email-address" autoCapitalize="none" />
      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
      <View style={{ height: spacing.md }} />
      <Button label={loading ? 'SENDING...' : 'SEND RESET LINK'} onPress={submit} disabled={loading} />
    </Screen>
  );
}
