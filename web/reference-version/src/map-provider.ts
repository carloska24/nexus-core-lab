export const MAP_PROVIDER = {
  name: 'OpenFreeMap',
  styleURL: 'https://tiles.openfreemap.org/styles/liberty',
  center: [-47.062, -22.868] as [number, number],
  zoom: 11.45,
  minZoom: 10,
  maxZoom: 16,
  attribution: 'OpenFreeMap · © OpenStreetMap contributors',
} as const;
