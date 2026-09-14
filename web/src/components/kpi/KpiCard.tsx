import React from 'react';
import { Users, Smartphone, Wifi, Radio, FileText } from 'lucide-react';
import { KpiCardProps } from '../../types';
import './Kpi.css';

export const KpiCard: React.FC<KpiCardProps> = ({
  label,
  value,
  change,
  badgeText,
  subtitle,
  iconName,
  accentColor = 'var(--color-blue)',
}) => {
  const isPurple = iconName === 'file-text';

  const renderIcon = () => {
    const iconColor = isPurple ? '#c084fc' : (accentColor || '#38bdf8');
    const props = { size: 20, style: { color: iconColor } };

    switch (iconName) {
      case 'users':
        return <Users {...props} />;
      case 'smartphone':
        return <Smartphone {...props} />;
      case 'wifi':
        return <Wifi {...props} />;
      case 'radio':
        return <Radio {...props} />;
      case 'file-text':
        return <FileText {...props} />;
      default:
        return <Users {...props} />;
    }
  };

  return (
    <div className="kpi-card">
      <div className="kpi-card-header">
        <div className={`kpi-icon-wrap ${isPurple ? 'purple-bg' : 'blue-bg'}`}>
          {renderIcon()}
        </div>

        {badgeText ? (
          <span className="kpi-badge-status">{badgeText}</span>
        ) : change ? (
          <span className={`kpi-badge-trend ${change.positive ? 'positive' : 'negative'}`}>
            <span className="trend-arrow">↑</span> {change.value}
          </span>
        ) : null}
      </div>

      <div className="kpi-body">
        <div className="kpi-label">{label}</div>
        <div className="kpi-value font-mono">{value}</div>
        <div className="kpi-subtitle font-mono">{subtitle}</div>
      </div>
    </div>
  );
};
