import React, { useState } from 'react';
import { View } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { useAuth } from '../../context/AuthContext';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';
import { spacing } from '../../theme';
import { AuthStackParams } from '../../navigation/AuthNavigator';
import { apiError } from '../../api/error';

type Nav = StackNavigationProp<AuthStackParams, 'Register'>;

export default function RegisterScreen() {
  const nav = useNavigation<Nav>();
  const { register } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function submit() {
    setError('');
    setLoading(true);
    try {
      await register(email, password, name);
      nav.navigate('VerifyEmail', { email });
    } catch (e: any) {
      setError(apiError(e, 'REGISTRATION FAILED'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Screen title="NEW ENTITY" scroll>
      <Input label="DISPLAY NAME" value={name} onChangeText={setName} autoCapitalize="none" />
      <Input label="EMAIL" value={email} onChangeText={setEmail} keyboardType="email-address" autoCapitalize="none" />
      <Input label="PASSWORD" value={password} onChangeText={setPassword} secureTextEntry />
      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
      <View style={{ height: spacing.md }} />
      <Button label={loading ? 'CREATING...' : 'CREATE ACCOUNT'} onPress={submit} disabled={loading} />
    </Screen>
  );
}
