import React from 'react';
import { Layers, ArrowRight } from 'lucide-react';
import { ActiveSessionRow } from '../../types';
import './Tables.css';

interface ActiveSessionsTableProps {
  sessions: ActiveSessionRow[];
  totalCount?: number;
}

export const ActiveSessionsTable: React.FC<ActiveSessionsTableProps> = ({
  sessions,
  totalCount = 17,
}) => {
  return (
    <div className="table-card">
      <div className="table-card-header">
        <div className="table-title-group">
          <div className="table-icon-wrap">
            <Layers size={17} className="text-cyan" />
          </div>
          <h3 className="table-title">
            Active Sessions <span className="title-counter">({totalCount})</span>
          </h3>
        </div>
        <a href="#all-sessions" className="view-all-link">
          <span>View all</span>
          <ArrowRight size={13} />
        </a>
      </div>

      <div className="table-responsive">
        <table className="dense-table">
          <thead>
            <tr>
              <th>Device ID</th>
              <th>Subscriber</th>
              <th>Cell</th>
              <th>IP Address</th>
              <th>Duration</th>
            </tr>
          </thead>
          <tbody>
            {sessions.map((sess) => (
              <tr key={sess.id}>
                <td className="font-mono device-cell">
                  <span className="row-led green" />
                  <span>{sess.deviceId}</span>
                </td>
                <td className="font-mono text-secondary">{sess.subscriber}</td>
                <td>
                  <span className="cell-tag">{sess.cell}</span>
                </td>
                <td className="font-mono text-primary">{sess.ipAddress}</td>
                <td className="font-mono text-muted">{sess.duration}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
