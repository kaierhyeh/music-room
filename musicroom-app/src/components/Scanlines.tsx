import React from 'react';
import { StyleSheet, View } from 'react-native';

export default function Scanlines() {
  return (
    <View style={StyleSheet.absoluteFill} pointerEvents="none">
      {Array.from({ length: 120 }).map((_, i) => (
        <View key={i} style={[styles.line, { top: i * 8 }]} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  line: {
    position: 'absolute',
    left: 0,
    right: 0,
    height: 1,
    backgroundColor: '#000',
    opacity: 0.18,
  },
});
