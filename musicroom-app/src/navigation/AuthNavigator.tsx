import React from 'react';
import { createStackNavigator } from '@react-navigation/stack';
import { colors } from '../theme';
import LoginScreen from '../screens/auth/LoginScreen';
import RegisterScreen from '../screens/auth/RegisterScreen';
import VerifyEmailScreen from '../screens/auth/VerifyEmailScreen';
import ForgotPasswordScreen from '../screens/auth/ForgotPasswordScreen';
import ResetPasswordScreen from '../screens/auth/ResetPasswordScreen';

export type AuthStackParams = {
  Login: undefined;
  Register: undefined;
  VerifyEmail: { email?: string };
  ForgotPassword: undefined;
  ResetPassword: undefined;
};

const Stack = createStackNavigator<AuthStackParams>();

export default function AuthNavigator() {
  return (
    <Stack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.bg },
        headerTintColor: colors.accent,
        headerTitleStyle: { fontFamily: 'VT323_400Regular', fontSize: 22, letterSpacing: 2 },
        cardStyle: { backgroundColor: colors.bg },
      }}
    >
      <Stack.Screen name="Login" component={LoginScreen} options={{ headerShown: false }} />
      <Stack.Screen name="Register" component={RegisterScreen} options={{ title: '// REGISTER' }} />
      <Stack.Screen name="VerifyEmail" component={VerifyEmailScreen} options={{ title: '// VERIFY EMAIL' }} />
      <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} options={{ title: '// FORGOT PASSWORD' }} />
      <Stack.Screen name="ResetPassword" component={ResetPasswordScreen} options={{ title: '// RESET PASSWORD' }} />
    </Stack.Navigator>
  );
}
