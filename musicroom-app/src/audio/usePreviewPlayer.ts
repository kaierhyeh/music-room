import { useEffect, useRef, useState } from 'react';
import {
  createAudioPlayer,
  setAudioModeAsync,
  type AudioPlayer,
  type AudioStatus,
} from 'expo-audio';
import { tracksApi } from '../api/tracks';
import { apiError } from '../api/error';

type Playable = { id: string; track_deezer_id: number };

type StatusEmitter = {
  addListener(event: 'playbackStatusUpdate', listener: (s: AudioStatus) => void): void;
};

let stopActive: (() => void) | null = null;

function release(sound: AudioPlayer | null) {
  if (!sound) return;
  sound.pause();
  sound.remove();
}

export function usePreviewPlayer(onFinish?: (id: string) => void) {
  const [playingId, setPlayingId] = useState<string | null>(null);
  const [loadingId, setLoadingId] = useState<string | null>(null);
  const [error, setError] = useState('');
  const soundRef = useRef<AudioPlayer | null>(null);
  const reqRef = useRef(0);
  const onFinishRef = useRef(onFinish);
  onFinishRef.current = onFinish;
  const stopRef = useRef(stop);
  stopRef.current = stop;
  const ownerRef = useRef(() => void stopRef.current());

  useEffect(() => {
    setAudioModeAsync({ playsInSilentMode: true }).catch(() => {});
    return () => {
      reqRef.current++;
      release(soundRef.current);
      if (stopActive === ownerRef.current) stopActive = null;
    };
  }, []);

  async function stop() {
    reqRef.current++;
    if (stopActive === ownerRef.current) stopActive = null;
    const sound = soundRef.current;
    soundRef.current = null;
    setPlayingId(null);
    setLoadingId(null);
    release(sound);
  }

  async function play(track: Playable) {
    await stop();
    stopActive?.();
    stopActive = ownerRef.current;
    const req = reqRef.current;
    setError('');
    setLoadingId(track.id);
    try {
      const url = await tracksApi.preview(track.track_deezer_id);
      if (req !== reqRef.current) return;
      const sound = createAudioPlayer({ uri: url });
      soundRef.current = sound;
      setPlayingId(track.id);
      (sound as unknown as StatusEmitter).addListener('playbackStatusUpdate', (s) => {
        if (!s.didJustFinish || req !== reqRef.current) return;
        stop().then(() => onFinishRef.current?.(track.id));
      });
      sound.play();
    } catch (e) {
      if (req !== reqRef.current) return;
      setError(apiError(e, 'THIS PREVIEW COULD NOT BE PLAYED. TRY ANOTHER TRACK.'));
    } finally {
      if (req === reqRef.current) setLoadingId(null);
    }
  }

  function toggle(track: Playable) {
    return playingId === track.id || loadingId === track.id ? stop() : play(track);
  }

  return { playingId, loadingId, error, play, stop, toggle };
}
