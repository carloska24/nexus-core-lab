import React from 'react';
import { FileText, ArrowRight } from 'lucide-react';
import { RecentEventRow, EventType } from '../../types';
import '../tables/Tables.css';

interface RecentEventsFeedProps {
  events: RecentEventRow[];
}

export const RecentEventsFeed: React.FC<RecentEventsFeedProps> = ({ events }) => {
  const getEventBadgeClass = (type: EventType) => {
    switch (type) {
      case 'ATTACH':
        return 'badge-attach';
      case 'CELL_HANDOVER':
        return 'badge-handover';
      case 'DETACH':
        return 'badge-detach';
      default:
        return 'badge-other';
    }
  };

  const getEventDotClass = (type: EventType) => {
    switch (type) {
      case 'ATTACH':
        return 'dot-green';
      case 'CELL_HANDOVER':
        return 'dot-purple';
      case 'DETACH':
        return 'dot-red';
      default:
        return 'dot-blue';
    }
  };

  return (
    <div className="table-card events-card">
      <div className="table-card-header">
        <div className="table-title-group">
          <div className="table-icon-wrap">
            <FileText size={17} className="text-purple" />
          </div>
          <h3 className="table-title">Recent Events</h3>
        </div>
        <a href="#all-events" className="view-all-link">
          <span>View all</span>
          <ArrowRight size={13} />
        </a>
      </div>

      <div className="table-responsive">
        <table className="dense-table events-table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Event Type</th>
              <th>Device</th>
              <th>Details</th>
            </tr>
          </thead>
          <tbody>
            {events.map((evt) => (
              <tr key={evt.id}>
                <td className="font-mono text-muted event-time-cell">
                  <span className={`event-dot ${getEventDotClass(evt.type)}`} />
                  <span>{evt.time}</span>
                </td>
                <td>
                  <span className={`event-type-badge font-mono ${getEventBadgeClass(evt.type)}`}>
                    {evt.type}
                  </span>
                </td>
                <td className="font-mono text-secondary">{evt.device}</td>
                <td className="text-secondary event-details">{evt.details}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
