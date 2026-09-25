import * as Linking from 'expo-linking';
import type { LinkingOptions } from '@react-navigation/native';
import type { AuthStackParams } from './AuthNavigator';
import type { AppTabParams } from './AppNavigator';

type RootParams = AuthStackParams & AppTabParams;

const linking: LinkingOptions<RootParams> = {
  prefixes: [Linking.createURL('/')],
  config: {
    screens: {
      Login: 'login',
      Register: 'register',
      VerifyEmail: 'verify-email',
      ForgotPassword: 'forgot-password',
      ResetPassword: 'reset-password',

      EventsTab: {
        path: 'events',
        initialRouteName: 'EventsList',
        screens: {
          EventsList: '',
          CreateEvent: 'new',
          EventDetail: ':eventId',
        },
      },
      PlaylistsTab: {
        path: 'playlists',
        initialRouteName: 'PlaylistsList',
        screens: {
          PlaylistsList: '',
          CreatePlaylist: 'new',
          EditPlaylist: 'edit/:playlistId',
          UserProfile: 'user/:userId',
          PlaylistDetail: ':playlistId',
        },
      },
      FriendsTab: {
        path: 'friends',
        initialRouteName: 'Friends',
        screens: {
          Friends: '',
          FriendRequests: 'requests',
          UserSearch: 'search',
          UserProfile: 'user/:userId',
        },
      },
      ProfileTab: {
        path: 'profile',
        initialRouteName: 'Profile',
        screens: {
          Profile: '',
          EditProfile: 'edit',
        },
      },
    },
  },
};

export default linking;
