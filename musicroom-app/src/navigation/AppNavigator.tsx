import React from 'react';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createStackNavigator } from '@react-navigation/stack';
import type { NavigatorScreenParams } from '@react-navigation/native';
import { Text } from 'react-native';
import { colors } from '../theme';
import { useResponsive } from '../theme/useResponsive';

import EventsListScreen from '../screens/events/EventsListScreen';
import CreateEventScreen from '../screens/events/CreateEventScreen';
import EventDetailScreen from '../screens/events/EventDetailScreen';
import PlaylistsListScreen from '../screens/playlists/PlaylistsListScreen';
import CreatePlaylistScreen from '../screens/playlists/CreatePlaylistScreen';
import EditPlaylistScreen from '../screens/playlists/EditPlaylistScreen';
import PlaylistDetailScreen from '../screens/playlists/PlaylistDetailScreen';
import FriendsScreen from '../screens/friends/FriendsScreen';
import FriendRequestsScreen from '../screens/friends/FriendRequestsScreen';
import UserSearchScreen from '../screens/friends/UserSearchScreen';
import ProfileScreen from '../screens/profile/ProfileScreen';
import EditProfileScreen from '../screens/profile/EditProfileScreen';
import UserProfileScreen from '../screens/profile/UserProfileScreen';

export type EventsStackParams = {
  EventsList: undefined;
  CreateEvent: undefined;
  EventDetail: { eventId: string };
};

export type PlaylistsStackParams = {
  PlaylistsList: { tab?: string } | undefined;
  CreatePlaylist: undefined;
  EditPlaylist: { playlistId: string };
  PlaylistDetail: { playlistId: string };
  UserProfile: { userId: string };
};

export type FriendsStackParams = {
  Friends: undefined;
  FriendRequests: undefined;
  UserSearch: undefined;
  UserProfile: { userId: string };
};

export type ProfileStackParams = {
  Profile: undefined;
  EditProfile: undefined;
};

export type AppTabParams = {
  EventsTab: NavigatorScreenParams<EventsStackParams>;
  PlaylistsTab: NavigatorScreenParams<PlaylistsStackParams>;
  FriendsTab: NavigatorScreenParams<FriendsStackParams>;
  ProfileTab: NavigatorScreenParams<ProfileStackParams>;
};

const Tab = createBottomTabNavigator<AppTabParams>();
const EventsNav = createStackNavigator<EventsStackParams>();
const PlaylistsNav = createStackNavigator<PlaylistsStackParams>();
const FriendsNav = createStackNavigator<FriendsStackParams>();
const ProfileNav = createStackNavigator<ProfileStackParams>();

const screenOpts = {
  headerStyle: { backgroundColor: colors.bg },
  headerTintColor: colors.accent,
  headerTitleStyle: { fontFamily: 'VT323_400Regular', fontSize: 22, letterSpacing: 2 },
  cardStyle: { backgroundColor: colors.bg },
};

function EventsStack() {
  return (
    <EventsNav.Navigator screenOptions={screenOpts}>
      <EventsNav.Screen name="EventsList" component={EventsListScreen} options={{ title: '// EVENTS' }} />
      <EventsNav.Screen name="CreateEvent" component={CreateEventScreen} options={{ title: '// NEW EVENT' }} />
      <EventsNav.Screen name="EventDetail" component={EventDetailScreen} options={{ title: '// EVENT' }} />
    </EventsNav.Navigator>
  );
}

function PlaylistsStack() {
  return (
    <PlaylistsNav.Navigator screenOptions={screenOpts}>
      <PlaylistsNav.Screen name="PlaylistsList" component={PlaylistsListScreen} options={{ title: '// PLAYLISTS' }} />
      <PlaylistsNav.Screen name="CreatePlaylist" component={CreatePlaylistScreen} options={{ title: '// NEW PLAYLIST' }} />
      <PlaylistsNav.Screen name="EditPlaylist" component={EditPlaylistScreen} options={{ title: '// EDIT PLAYLIST' }} />
      <PlaylistsNav.Screen name="PlaylistDetail" component={PlaylistDetailScreen} options={{ title: '// PLAYLIST' }} />
      <PlaylistsNav.Screen name="UserProfile" component={UserProfileScreen} options={{ title: '// PROFILE' }} />
    </PlaylistsNav.Navigator>
  );
}

function FriendsStack() {
  return (
    <FriendsNav.Navigator screenOptions={screenOpts}>
      <FriendsNav.Screen name="Friends" component={FriendsScreen} options={{ title: '// FRIENDS' }} />
      <FriendsNav.Screen name="FriendRequests" component={FriendRequestsScreen} options={{ title: '// REQUESTS' }} />
      <FriendsNav.Screen name="UserSearch" component={UserSearchScreen} options={{ title: '// FIND USER' }} />
      <FriendsNav.Screen name="UserProfile" component={UserProfileScreen} options={{ title: '// PROFILE' }} />
    </FriendsNav.Navigator>
  );
}

function ProfileStack() {
  return (
    <ProfileNav.Navigator screenOptions={screenOpts}>
      <ProfileNav.Screen name="Profile" component={ProfileScreen} options={{ title: '// PROFILE' }} />
      <ProfileNav.Screen name="EditProfile" component={EditProfileScreen} options={{ title: '// EDIT PROFILE' }} />
    </ProfileNav.Navigator>
  );
}

const tabIcon = (label: string) =>
  ({ focused }: { focused: boolean }) =>
    <Text style={{ fontFamily: 'VT323_400Regular', fontSize: 10, color: focused ? colors.accent : colors.textDim, letterSpacing: 1 }}>{label}</Text>;

export default function AppNavigator() {
  const { wide } = useResponsive();

  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
        tabBarPosition: wide ? 'left' : 'bottom',
        tabBarStyle: wide
          ? {
              backgroundColor: colors.bg,
              borderRightColor: colors.borderAccent,
              borderRightWidth: 1,
              width: 168,
              paddingTop: 12,
            }
          : {
              backgroundColor: colors.bg,
              borderTopColor: colors.borderAccent,
              borderTopWidth: 1,
              height: 56,
            },
        tabBarActiveTintColor: colors.accent,
        tabBarInactiveTintColor: colors.textDim,
        tabBarActiveBackgroundColor: wide ? colors.accentGlow : 'transparent',
        tabBarInactiveBackgroundColor: 'transparent',
        tabBarLabelStyle: { fontFamily: 'VT323_400Regular', fontSize: 14, letterSpacing: 1 },
      }}
    >
      <Tab.Screen
        name="EventsTab"
        component={EventsStack}
        options={{ tabBarLabel: 'EVENTS', tabBarIcon: tabIcon('♪') }}
      />
      <Tab.Screen
        name="PlaylistsTab"
        component={PlaylistsStack}
        options={{ tabBarLabel: 'PLAYLISTS', tabBarIcon: tabIcon('≡') }}
      />
      <Tab.Screen
        name="FriendsTab"
        component={FriendsStack}
        options={{ tabBarLabel: 'FRIENDS', tabBarIcon: tabIcon('▲') }}
      />
      <Tab.Screen
        name="ProfileTab"
        component={ProfileStack}
        options={{ tabBarLabel: 'PROFILE', tabBarIcon: tabIcon('□') }}
      />
    </Tab.Navigator>
  );
}
