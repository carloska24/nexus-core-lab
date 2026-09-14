import React from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { Activity } from 'lucide-react';
import { SessionActivityPoint } from '../../types';
import './BottomAnalytics.css';

interface SessionActivityChartProps {
  data: SessionActivityPoint[];
}

export const SessionActivityChart: React.FC<SessionActivityChartProps> = ({ data }) => {
  return (
    <div className="analytics-card activity-card">
      <div className="analytics-header">
        <div className="analytics-title-wrap">
          <div className="analytics-icon-wrap blue-bg">
            <Activity size={16} className="text-cyan" />
          </div>
          <div>
            <h3 className="analytics-title">Session Activity</h3>
            <div className="analytics-subtitle">Active vs Detached sessions (last 30 minutes)</div>
          </div>
        </div>

        {/* Chart Legend */}
        <div className="chart-legend-top">
          <div className="legend-entry">
            <span className="legend-indicator green" />
            <span>Active Sessions</span>
          </div>
          <div className="legend-entry">
            <span className="legend-indicator blue" />
            <span>Detached Sessions</span>
          </div>
        </div>
      </div>

      {/* Chart Canvas */}
      <div className="chart-canvas-container">
        <ResponsiveContainer width="100%" height={125}>
          <AreaChart data={data} margin={{ top: 8, right: 10, left: -25, bottom: 0 }}>
            <defs>
              <linearGradient id="colorActive" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#10b981" stopOpacity={0.35} />
                <stop offset="95%" stopColor="#10b981" stopOpacity={0.0} />
              </linearGradient>
              <linearGradient id="colorDetached" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#141f30" vertical={false} />
            <XAxis
              dataKey="time"
              stroke="#64748b"
              fontSize={10}
              tickLine={false}
              axisLine={{ stroke: '#1a2336' }}
            />
            <YAxis
              stroke="#64748b"
              fontSize={10}
              tickLine={false}
              axisLine={false}
              domain={[0, 40]}
              ticks={[0, 10, 20, 30, 40]}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: '#070b14',
                borderColor: '#1a2538',
                borderRadius: '6px',
                fontSize: '11px',
                color: '#ffffff',
              }}
            />
            <Area
              type="monotone"
              dataKey="activeSessions"
              stroke="#10b981"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#colorActive)"
              dot={{ r: 2.5, fill: '#10b981' }}
              isAnimationActive={false}
            />
            <Area
              type="monotone"
              dataKey="detachedSessions"
              stroke="#38bdf8"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#colorDetached)"
              dot={{ r: 2.5, fill: '#38bdf8' }}
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};
