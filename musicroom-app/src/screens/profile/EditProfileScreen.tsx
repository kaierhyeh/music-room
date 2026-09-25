import React, { useState } from 'react';
import { View, StyleSheet } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { useAuth } from '../../context/AuthContext';
import { usersApi, FieldVisibility } from '../../api/users';
import Screen from '../../components/ui/Screen';
import Text from '../../components/ui/Text';
import Input from '../../components/ui/Input';
import Button from '../../components/ui/Button';
import { colors, spacing } from '../../theme';
import { apiError } from '../../api/error';

const VIS_OPTIONS: FieldVisibility[] = ['public', 'friends', 'private'];
const VIS_LABELS: Record<FieldVisibility, string> = { public: 'PUB', friends: 'FRN', private: 'PVT' };

const GENRES = [
  'rock', 'pop', 'hip-hop', 'rap', 'electro', 'house',
  'techno', 'jazz', 'blues', 'soul', 'funk', 'r&b',
  'metal', 'punk', 'indie', 'classical', 'reggae', 'latin',
];

function VisPicker({ value, onChange }: { value: FieldVisibility; onChange: (v: FieldVisibility) => void }) {
  return (
    <View style={styles.visPicker}>
      {VIS_OPTIONS.map((v) => (
        <Button
          key={v}
          label={VIS_LABELS[v]}
          onPress={() => onChange(v)}
          variant={value === v ? 'primary' : 'ghost'}
        />
      ))}
    </View>
  );
}

function parseArtists(raw: string): string[] {
  const seen = new Set<string>();
  return raw
    .split(',')
    .map((a) => a.trim())
    .filter((a) => {
      const key = a.toLowerCase();
      if (!a || seen.has(key)) return false;
      seen.add(key);
      return true;
    });
}

export default function EditProfileScreen() {
  const nav = useNavigation<any>();
  const { user, refreshUser } = useAuth();
  const [name, setName] = useState(user?.display_name ?? '');
  const [bio, setBio] = useState(user?.profile_data?.bio ?? '');
  const [location, setLocation] = useState(user?.profile_data?.location ?? '');
  const [genres, setGenres] = useState<string[]>(user?.music_preferences?.genres ?? []);
  const [artists, setArtists] = useState((user?.music_preferences?.artists ?? []).join(', '));
  const [vis, setVis] = useState<Record<string, FieldVisibility>>({
    bio: 'public',
    location: 'public',
    music_preferences: 'public',
    ...user?.profile_visibility,
  });
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  function setFieldVis(field: string, v: FieldVisibility) {
    setVis((prev) => ({ ...prev, [field]: v }));
  }

  function toggleGenre(g: string) {
    setGenres((prev) => (prev.includes(g) ? prev.filter((x) => x !== g) : [...prev, g]));
  }

  async function save() {
    setError('');
    setLoading(true);
    try {
      await usersApi.update({
        display_name: name,
        profile_data: { bio, location },
        profile_visibility: vis,
        music_preferences: { genres, artists: parseArtists(artists) },
      });
      await refreshUser();
      nav.goBack();
    } catch (e: any) {
      setError(apiError(e, 'COULD NOT SAVE YOUR CHANGES. TRY AGAIN.'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Screen title="EDIT PROFILE" scroll>
      <Input label="DISPLAY NAME" value={name} onChangeText={setName} />

      <View style={styles.divider} />
      <Text size="sm" dim>{'// BIO'}</Text>
      <VisPicker value={vis.bio} onChange={(v) => setFieldVis('bio', v)} />
      <Input value={bio} onChangeText={setBio} multiline numberOfLines={3} />

      <View style={styles.divider} />
      <Text size="sm" dim>{'// LOCATION'}</Text>
      <VisPicker value={vis.location} onChange={(v) => setFieldVis('location', v)} />
      <Input value={location} onChangeText={setLocation} />

      <View style={styles.divider} />
      <Text size="sm" dim>{'// MUSIC PREFERENCES'}</Text>
      <VisPicker
        value={vis.music_preferences}
        onChange={(v) => setFieldVis('music_preferences', v)}
      />
      <Text size="xs" dim>{'// GENRES'}</Text>
      <View style={styles.chips}>
        {GENRES.map((g) => (
          <Button
            key={g}
            label={g}
            onPress={() => toggleGenre(g)}
            variant={genres.includes(g) ? 'primary' : 'ghost'}
          />
        ))}
      </View>
      <Input
        label="FAVOURITE ARTISTS (COMMA SEPARATED)"
        value={artists}
        onChangeText={setArtists}
        autoCapitalize="words"
        multiline
        numberOfLines={2}
      />

      {error ? <Text size="sm" danger>{`! ${error}`}</Text> : null}
      <View style={{ height: spacing.md }} />
      <Button label={loading ? 'SAVING...' : 'SAVE'} onPress={save} disabled={loading} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  divider: { height: 1, backgroundColor: colors.border, marginVertical: spacing.sm },
  visPicker: { flexDirection: 'row', gap: 8 },
  chips: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
});
