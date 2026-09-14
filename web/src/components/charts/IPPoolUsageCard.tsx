import React from 'react';
import { Database, CheckCircle2 } from 'lucide-react';
import { IPPoolUsageData } from '../../types';
import './BottomAnalytics.css';

interface IPPoolUsageCardProps {
  data: IPPoolUsageData;
}

export const IPPoolUsageCard: React.FC<IPPoolUsageCardProps> = ({ data }) => {
  const available = data.total - data.used;

  return (
    <div className="analytics-card ip-pool-card">
      <div className="analytics-header">
        <div className="analytics-title-wrap">
          <div className="analytics-icon-wrap blue-bg">
            <Database size={16} className="text-cyan" />
          </div>
          <div>
            <h3 className="analytics-title">IP Pool Usage</h3>
            <span className="analytics-subtitle font-mono">{data.subnet} {data.typeLabel}</span>
          </div>
        </div>
      </div>

      <div className="ip-pool-content">
        <div className="ip-pool-stat-row">
          <span className="ip-pool-left-label font-mono">
            <strong>{data.used}</strong> / {data.total} IPs in use
          </span>
          <span className="ip-pool-right-percent font-mono">
            {data.percent}%
          </span>
        </div>

        <div className="ip-progress-track">
          <div
            className="ip-progress-fill green-bar"
            style={{ width: `${Math.min(data.percent, 100)}%` }}
          />
        </div>

        <div className="ip-pool-legend">
          <span className="ip-legend-item">
            <span className="ip-dot dot-green" />
            <span>In Use ({data.used})</span>
          </span>
          <span className="ip-legend-item">
            <span className="ip-dot dot-gray" />
            <span>Available ({available})</span>
          </span>
        </div>

        {data.warmUpStatus && (
          <div className="ip-warmup-box">
            <CheckCircle2 size={15} className="warmup-check-icon" />
            <div className="warmup-text-group">
              <span className="warmup-title">{data.warmUpStatus.title}</span>
              <span className="warmup-subtitle">{data.warmUpStatus.subtitle}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
