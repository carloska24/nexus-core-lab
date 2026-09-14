import React, { useState } from 'react';
import { MapPin, Plus, Minus, Radio } from 'lucide-react';
import { NetworkCell, ConnectedDeviceNode } from '../../types';
import './NetworkTopology.css';

interface NetworkTopologyProps {
  cells: NetworkCell[];
  devices: ConnectedDeviceNode[];
}

export const NetworkTopology: React.FC<NetworkTopologyProps> = ({ cells, devices }) => {
  const [zoomLevel, setZoomLevel] = useState(1);

  const handleZoomIn = () => setZoomLevel((z) => Math.min(Number((z + 0.15).toFixed(2)), 1.45));
  const handleZoomOut = () => setZoomLevel((z) => Math.max(Number((z - 0.15).toFixed(2)), 0.85));

  const cellMap = new Map(cells.map((c) => [c.id, c]));

  return (
    <div className="topology-card">
      {/* Topology Header */}
      <div className="topology-header">
        <div className="topology-title-wrap">
          <div className="topology-icon">
            <Radio size={17} className="text-cyan" />
          </div>
          <div>
            <h2 className="topology-title">Network Topology</h2>
            <div className="topology-subtitle">Live view of cells and connected devices</div>
          </div>
        </div>

        {/* Legend */}
        <div className="topology-legend">
          <div className="legend-item">
            <span className="legend-dot lte" />
            <span>LTE Cell</span>
          </div>
          <div className="legend-item">
            <span className="legend-dot g5" />
            <span>5G Cell</span>
          </div>
          <div className="legend-item">
            <span className="legend-dot device" />
            <span>Connected Device</span>
          </div>
          <div className="legend-item">
            <span className="legend-dash handover" />
            <span>Handover</span>
          </div>
        </div>
      </div>

      {/* SVG Canvas Map Container */}
      <div className="topology-canvas-wrap">
        <svg
          className="topology-svg"
          viewBox="0 0 780 440"
          preserveAspectRatio="xMidYMid meet"
          style={{ transform: `scale(${zoomLevel})` }}
        >
          <defs>
            {/* Glow Filters */}
            <filter id="glow-cyan" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="5" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>

            <filter id="glow-purple" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="5" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>

            <filter id="glow-green" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="4" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>

            <filter id="glow-orange" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="4" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>

            {/* Radial Vignette for Urban NOC Map */}
            <radialGradient id="mapVignette" cx="50%" cy="50%" r="65%">
              <stop offset="0%" stopColor="#080e1a" stopOpacity="0.1" />
              <stop offset="60%" stopColor="#050912" stopOpacity="0.5" />
              <stop offset="100%" stopColor="#03060c" stopOpacity="0.9" />
            </radialGradient>
          </defs>

          {/* =================================================================
              1. NOCTURNAL CITY MAP BACKGROUND (CAMPINAS URBAN ARTERIES)
              ================================================================= */}
          <image
            href="/campinas-noc-map.jpg"
            x="0"
            y="0"
            width="780"
            height="440"
            preserveAspectRatio="xMidYMid slice"
            opacity="0.88"
          />

          {/* Vignette Overlay to blend with dark NOC dashboard theme */}
          <rect width="780" height="440" fill="url(#mapVignette)" pointerEvents="none" />

          {/* City Watermark */}
          <text
            x="390"
            y="215"
            textAnchor="middle"
            className="city-watermark font-mono"
          >
            CAMPINAS / SP
          </text>

          {/* =================================================================
              2. HANDOVER TRANSITION ARC (CELL-SP-001 -> CELL-SP-003)
              ================================================================= */}
          <g className="handover-layer">
            {/* Soft background glow path */}
            <path
              d="M 270 310 Q 410 230 550 310"
              fill="none"
              stroke="#f97316"
              strokeWidth="4"
              strokeOpacity="0.25"
              filter="url(#glow-orange)"
            />
            {/* Crisp dashed handover arc */}
            <path
              d="M 270 310 Q 410 230 550 310"
              fill="none"
              stroke="#f97316"
              strokeWidth="2.2"
              strokeDasharray="6 4"
            />
            {/* Handover Pill Badge */}
            <g transform="translate(410, 245)">
              <rect
                x="-42"
                y="-10"
                width="84"
                height="20"
                rx="10"
                fill="#070b14"
                stroke="#f97316"
                strokeWidth="1.2"
                filter="url(#glow-orange)"
              />
              <text
                x="0"
                y="3.5"
                textAnchor="middle"
                fill="#f97316"
                fontSize="10"
                fontWeight="700"
                letterSpacing="0.08em"
                className="font-mono"
              >
                HANDOVER
              </text>
            </g>
          </g>

          {/* =================================================================
              3. DEVICE CONNECTION LINKS (GREEN DASHED WIRES)
              ================================================================= */}
          <g className="device-links" stroke="#10b981" strokeWidth="1.2" strokeDasharray="3 3" opacity="0.8">
            {devices.map((dev) => {
              const cell = cellMap.get(dev.cellId);
              if (!cell) return null;
              return (
                <line
                  key={`link-${dev.id}-${cell.id}`}
                  x1={dev.x}
                  y1={dev.y}
                  x2={cell.x}
                  y2={cell.y}
                />
              );
            })}
          </g>

          {/* =================================================================
              4. CELL TOWERS WITH RICH ANTENNA MASTS & GLOWING HALOS
              ================================================================= */}
          <g className="cell-towers">
            {cells.map((cell) => {
              const is5G = cell.tech === '5G';
              const mainColor = is5G ? '#c084fc' : '#38bdf8';
              const glowFilter = is5G ? 'url(#glow-purple)' : 'url(#glow-cyan)';
              const pulseFill = is5G ? 'rgba(192, 132, 252, 0.14)' : 'rgba(56, 189, 248, 0.14)';

              return (
                <g key={cell.id} transform={`translate(${cell.x}, ${cell.y})`} className="cell-node">
                  {/* Outer Transmission Pulse Wave Ring */}
                  <circle
                    r="44"
                    fill="none"
                    stroke={mainColor}
                    strokeWidth="1"
                    strokeDasharray="4 4"
                    strokeOpacity="0.45"
                  />
                  {/* Mid Energy Glow Circle */}
                  <circle
                    r="28"
                    fill={pulseFill}
                    stroke={mainColor}
                    strokeWidth="1.5"
                    filter={glowFilter}
                  />

                  {/* Tower Mast Icon Graphic */}
                  <g className="tower-icon-graphic">
                    {/* Radiating Waves on Top */}
                    <path
                      d="M -6 -13 A 8 8 0 0 1 6 -13"
                      fill="none"
                      stroke={mainColor}
                      strokeWidth="1.6"
                    />
                    <path
                      d="M -10 -16 A 13 13 0 0 1 10 -16"
                      fill="none"
                      stroke={mainColor}
                      strokeWidth="1.4"
                      strokeOpacity="0.75"
                    />
                    <path
                      d="M -14 -19 A 18 18 0 0 1 14 -19"
                      fill="none"
                      stroke={mainColor}
                      strokeWidth="1.2"
                      strokeOpacity="0.5"
                    />
                    {/* Mast Spire & Lattice Tripod */}
                    <line x1="0" y1="-10" x2="0" y2="12" stroke="#ffffff" strokeWidth="1.6" />
                    <line x1="-8" y1="12" x2="0" y2="-6" stroke={mainColor} strokeWidth="1.4" />
                    <line x1="8" y1="12" x2="0" y2="-6" stroke={mainColor} strokeWidth="1.4" />
                    <line x1="-5" y1="3" x2="5" y2="3" stroke={mainColor} strokeWidth="1.2" />
                    <line x1="-7" y1="8" x2="7" y2="8" stroke={mainColor} strokeWidth="1.2" />
                    {/* Emitter Tip Beacon */}
                    <circle cx="0" cy="-9" r="2.5" fill="#ffffff" filter={glowFilter} />
                  </g>

                  {/* Cell Label & Tech Pill */}
                  <g transform="translate(0, 32)">
                    <text
                      textAnchor="middle"
                      className="cell-id-text"
                      fill="#ffffff"
                      fontSize="12"
                      fontWeight="700"
                    >
                      {cell.id}
                    </text>
                    <text
                      y="14"
                      textAnchor="middle"
                      className="cell-tech-text"
                      fill={mainColor}
                      fontSize="11"
                      fontWeight="800"
                      letterSpacing="0.08em"
                    >
                      {cell.tech}
                    </text>
                  </g>
                </g>
              );
            })}
          </g>

          {/* =================================================================
              5. CONNECTED DEVICE NODES (EMERALD GREEN GLOWING UEs)
              ================================================================= */}
          <g className="connected-devices">
            {devices.map((dev) => (
              <g key={dev.id} transform={`translate(${dev.x}, ${dev.y})`} className="device-node">
                {/* Green Outer Glow Halo */}
                <circle r="6" fill="#10b981" filter="url(#glow-green)" />
                {/* Core White Pinpoint */}
                <circle r="2.5" fill="#ffffff" />
                {/* Device Label */}
                <text
                  y="15"
                  textAnchor="middle"
                  className="device-id-text font-mono"
                  fill="#ffffff"
                  fontSize="11"
                  fontWeight="600"
                >
                  {dev.id}
                </text>
              </g>
            ))}
          </g>

          {/* =================================================================
              6. NORTH COMPASS ROSE
              ================================================================= */}
          <g className="compass-rose" transform="translate(735, 45)">
            <circle cx="0" cy="0" r="14" fill="#0b121e" stroke="#1a273c" strokeWidth="1" />
            <polygon points="0,-11 -4,0 4,0" fill="#38bdf8" />
            <polygon points="0,11 -4,0 4,0" fill="#334155" />
            <text x="0" y="-15" textAnchor="middle" className="compass-n">
              N
            </text>
          </g>
        </svg>

        {/* Map Bottom Metadata & Zoom Controls */}
        <div className="topology-overlay-bottom">
          <div className="location-tag">
            <MapPin size={13} className="text-cyan" />
            <span>Campinas, SP - Brazil</span>
          </div>

          <div className="map-zoom-controls">
            <button className="zoom-btn" onClick={handleZoomOut} aria-label="Zoom Out">
              <Minus size={13} />
            </button>
            <button className="zoom-btn" onClick={handleZoomIn} aria-label="Zoom In">
              <Plus size={13} />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
