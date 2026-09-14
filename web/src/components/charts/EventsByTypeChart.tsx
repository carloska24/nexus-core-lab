import React from 'react';
import { PieChart, Pie, Cell, ResponsiveContainer } from 'recharts';
import { BarChart2 } from 'lucide-react';
import { EventTypeDistribution } from '../../types';
import './BottomAnalytics.css';

interface EventsByTypeChartProps {
  data: EventTypeDistribution[];
  totalEvents?: number;
}

export const EventsByTypeChart: React.FC<EventsByTypeChartProps> = ({
  data,
  totalEvents = 842,
}) => {
  return (
    <div className="analytics-card events-type-card">
      <div className="analytics-header">
        <div className="analytics-title-wrap">
          <div className="analytics-icon-wrap blue-bg">
            <BarChart2 size={16} className="text-cyan" />
          </div>
          <div>
            <h3 className="analytics-title">Events by Type</h3>
            <div className="analytics-subtitle">Distribution of network events</div>
          </div>
        </div>
      </div>

      {/* Donut Chart & Side Legend */}
      <div className="donut-content-layout">
        {/* Pie Canvas with Central Counter */}
        <div className="donut-chart-wrap">
          <ResponsiveContainer width={125} height={125}>
            <PieChart>
              <Pie
                data={data}
                cx="50%"
                cy="50%"
                innerRadius={38}
                outerRadius={56}
                paddingAngle={2.5}
                dataKey="count"
                stroke="none"
                isAnimationActive={false}
              >
                {data.map((entry) => (
                  <Cell key={entry.name} fill={entry.color} />
                ))}
              </Pie>
            </PieChart>
          </ResponsiveContainer>

          {/* Center Text inside Donut */}
          <div className="donut-center-info">
            <span className="donut-total font-mono">{totalEvents}</span>
            <span className="donut-label">Events</span>
          </div>
        </div>

        {/* Breakdown Legend on Right */}
        <div className="donut-legend-list">
          {data.map((item) => (
            <div key={item.name} className="donut-legend-row">
              <div className="legend-name-group">
                <span className="legend-bullet" style={{ backgroundColor: item.color }} />
                <span className="legend-item-name font-mono">{item.name}</span>
              </div>
              <span className="legend-item-percent font-mono">{item.percent}%</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
