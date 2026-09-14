import React from 'react';
import {
  Home,
  Users,
  Smartphone,
  Share2,
  Radio,
  ListFilter,
  TrendingUp,
  PlayCircle,
  Settings,
  Power,
  Database,
  Sliders,
} from 'lucide-react';
import './Sidebar.css';

export const Sidebar: React.FC = () => {
  return (
    <aside className="sidebar">
      {/* Brand Header */}
      <div className="sidebar-brand">
        <div className="brand-header-text">
          <div className="brand-title">NEXUS</div>
          <div className="brand-subtitle">CORE LAB</div>
        </div>
      </div>

      {/* Navigation Group */}
      <div className="sidebar-nav">
        <ul className="nav-list">
          <li className="nav-item active">
            <a href="#overview">
              <Home size={17} />
              <span>Overview</span>
            </a>
          </li>
          <li className="nav-item">
            <a href="#subscribers">
              <Users size={17} />
              <span>Subscribers</span>
            </a>
          </li>
          <li className="nav-item">
            <a href="#devices">
              <Smartphone size={17} />
              <span>Devices</span>
            </a>
          </li>
          <li className="nav-item">
            <a href="#sessions">
              <Share2 size={17} />
              <span>Sessions</span>
            </a>
          </li>
          <li className="nav-item">
            <a href="#network">
              <Radio size={17} />
              <span>Network</span>
            </a>
          </li>
          <li className="nav-item">
            <a href="#events">
              <ListFilter size={17} />
              <span>Events</span>
            </a>
          </li>
          <li className="nav-item">
            <a href="#telemetry">
              <TrendingUp size={17} />
              <span>Telemetry</span>
            </a>
          </li>
          <li className="nav-item">
            <a href="#simulator">
              <PlayCircle size={17} />
              <span>Simulator</span>
            </a>
          </li>
        </ul>

        {/* System & Status Group */}
        <div className="nav-section-title">System</div>
        <ul className="nav-list system-nav">
          <li className="nav-item">
            <a href="#system">
              <Settings size={17} />
              <span>System</span>
            </a>
          </li>
          <li className="nav-item status-item">
            <div className="status-label-wrap">
              <Power size={17} />
              <span>API Status</span>
            </div>
            <span className="status-led green" title="Mock Online Status" />
          </li>
          <li className="nav-item status-item">
            <div className="status-label-wrap">
              <Database size={17} />
              <span>Database</span>
            </div>
            <span className="status-led green" title="Mock Healthy Status" />
          </li>
          <li className="nav-item">
            <a href="#configuration">
              <Sliders size={17} />
              <span>Configuration</span>
            </a>
          </li>
        </ul>
      </div>

      {/* Decorative Campinas Night Skyline Footer */}
      <div className="sidebar-footer">
        <div className="skyline-art-container">
          <svg
            className="skyline-svg"
            viewBox="0 0 220 85"
            preserveAspectRatio="none"
          >
            <defs>
              <linearGradient id="skylineGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#1e3a5f" stopOpacity="0.8" />
                <stop offset="40%" stopColor="#101d33" stopOpacity="0.95" />
                <stop offset="100%" stopColor="#05080e" stopOpacity="1" />
              </linearGradient>
              <linearGradient id="glowWindow" x1="0" y1="0" x2="1" y2="1">
                <stop offset="0%" stopColor="#fef08a" stopOpacity="0.9" />
                <stop offset="100%" stopColor="#f59e0b" stopOpacity="0.6" />
              </linearGradient>
            </defs>

            {/* Distant Haze / Sky Glow */}
            <rect width="220" height="85" fill="url(#skylineGrad)" />

            {/* Back Layer Silhouettes (Tall Tower Blocks) */}
            <rect x="15" y="25" width="22" height="60" fill="#0c1729" />
            <line x1="26" y1="12" x2="26" y2="25" stroke="#38bdf8" strokeWidth="1" />
            <circle cx="26" cy="11" r="1.5" fill="#ef4444" />

            <rect x="55" y="18" width="26" height="67" fill="#0e1b30" />
            <polygon points="55,18 68,6 81,18" fill="#142642" />
            <line x1="68" y1="2" x2="68" y2="6" stroke="#38bdf8" strokeWidth="1" />
            <circle cx="68" cy="1" r="1.5" fill="#ef4444" />

            <rect x="105" y="15" width="30" height="70" fill="#0f1e36" />
            <rect x="150" y="28" width="24" height="57" fill="#0d182b" />
            <line x1="162" y1="16" x2="162" y2="28" stroke="#38bdf8" strokeWidth="1" />
            <circle cx="162" cy="15" r="1.5" fill="#ef4444" />

            <rect x="185" y="22" width="25" height="63" fill="#0a1424" />

            {/* Front Layer Buildings */}
            <rect x="5" y="45" width="18" height="40" fill="#080e1a" />
            <rect x="32" y="38" width="28" height="47" fill="#070c17" />
            <rect x="75" y="42" width="35" height="43" fill="#060b14" />
            <rect x="125" y="35" width="32" height="50" fill="#070c17" />
            <rect x="170" y="44" width="22" height="41" fill="#080e1a" />

            {/* Lighted Windows in Campinas Skyline */}
            {/* Tower 1 */}
            <rect x="19" y="32" width="3" height="3" fill="#fef08a" opacity="0.85" />
            <rect x="27" y="32" width="3" height="3" fill="#fef08a" opacity="0.75" />
            <rect x="19" y="40" width="3" height="3" fill="#38bdf8" opacity="0.8" />
            <rect x="27" y="48" width="3" height="3" fill="#fef08a" opacity="0.9" />

            {/* Central Tower */}
            <rect x="60" y="24" width="3" height="3" fill="#fef08a" opacity="0.9" />
            <rect x="68" y="24" width="3" height="3" fill="#38bdf8" opacity="0.85" />
            <rect x="60" y="32" width="3" height="3" fill="#fef08a" opacity="0.7" />
            <rect x="73" y="32" width="3" height="3" fill="#fef08a" opacity="0.85" />

            {/* Main Skyscraper */}
            <rect x="110" y="22" width="3" height="4" fill="#fef08a" opacity="0.85" />
            <rect x="118" y="22" width="3" height="4" fill="#38bdf8" opacity="0.9" />
            <rect x="126" y="22" width="3" height="4" fill="#fef08a" opacity="0.75" />
            <rect x="110" y="30" width="3" height="4" fill="#38bdf8" opacity="0.7" />
            <rect x="118" y="30" width="3" height="4" fill="#fef08a" opacity="0.9" />
            <rect x="126" y="30" width="3" height="4" fill="#fef08a" opacity="0.8" />
            <rect x="110" y="38" width="3" height="4" fill="#fef08a" opacity="0.9" />

            {/* East Tower */}
            <rect x="155" y="36" width="3" height="3" fill="#38bdf8" opacity="0.8" />
            <rect x="163" y="36" width="3" height="3" fill="#fef08a" opacity="0.85" />
            <rect x="155" y="44" width="3" height="3" fill="#fef08a" opacity="0.75" />
            <rect x="190" y="30" width="3" height="3" fill="#fef08a" opacity="0.8" />
            <rect x="198" y="30" width="3" height="3" fill="#38bdf8" opacity="0.75" />
            <rect x="198" y="38" width="3" height="3" fill="#fef08a" opacity="0.85" />

            {/* Horizon Bottom Blend */}
            <rect x="0" y="70" width="220" height="15" fill="url(#skylineGrad)" opacity="0.9" />
          </svg>
        </div>

        <div className="footer-content">
          <div className="footer-title">NEXUS CORE LAB</div>
          <div className="footer-location">CAMPINAS / SP</div>
          <div className="footer-slogan">"Connecting Ideas to a Smarter Tomorrow"</div>
        </div>
      </div>
    </aside>
  );
};
