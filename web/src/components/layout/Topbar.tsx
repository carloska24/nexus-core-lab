import React, { useState, useEffect } from 'react';
import { Settings } from 'lucide-react';
import './Topbar.css';

export const Topbar: React.FC = () => {
  const [timeString, setTimeString] = useState('10 Dez 2024  23:41:27');

  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      const day = String(now.getDate()).padStart(2, '0');
      const months = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];
      const month = months[now.getMonth()];
      const year = now.getFullYear();
      const hours = String(now.getHours()).padStart(2, '0');
      const minutes = String(now.getMinutes()).padStart(2, '0');
      const seconds = String(now.getSeconds()).padStart(2, '0');
      setTimeString(`${day} ${month} ${year}  ${hours}:${minutes}:${seconds}`);
    };

    updateTime();
    const interval = setInterval(updateTime, 1000);
    return () => clearInterval(interval);
  }, []);

  return (
    <header className="topbar">
      {/* Title & Slogan */}
      <div className="topbar-title-section">
        <h1 className="topbar-title">Telecom Network Simulator &amp; Core Lab</h1>
        <div className="topbar-subline">
          <span>Build</span>
          <span className="dot">·</span>
          <span>Learn</span>
          <span className="dot">·</span>
          <span>Simulate</span>
          <span className="dot">·</span>
          <span>Explore</span>
        </div>
      </div>

      {/* Right Controls & Health Indicators */}
      <div className="topbar-actions">
        {/* System Online Badge */}
        <div className="status-badge-online">
          <span className="online-dot" />
          <span className="online-text">SYSTEM ONLINE</span>
        </div>

        {/* Real-time Clock */}
        <div className="topbar-clock font-mono">
          {timeString}
        </div>

        {/* Settings Button */}
        <button className="topbar-btn" aria-label="Settings">
          <Settings size={18} />
        </button>

        {/* Core Infra Pill */}
        <div className="infra-pill">
          <span className="infra-item">
            API <span className="led green" />
          </span>
          <span className="infra-item">
            Database <span className="led green" />
          </span>
        </div>
      </div>
    </header>
  );
};
