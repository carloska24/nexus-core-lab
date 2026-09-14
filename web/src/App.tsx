import React from 'react';
import { Sidebar } from './components/layout/Sidebar';
import { Topbar } from './components/layout/Topbar';
import { KpiGrid } from './components/kpi/KpiGrid';
import { NetworkTopology } from './components/topology/NetworkTopology';
import { ActiveSessionsTable } from './components/tables/ActiveSessionsTable';
import { RecentEventsFeed } from './components/events/RecentEventsFeed';
import { SessionActivityChart } from './components/charts/SessionActivityChart';
import { EventsByTypeChart } from './components/charts/EventsByTypeChart';
import { IPPoolUsageCard } from './components/charts/IPPoolUsageCard';

import {
  mockKpis,
  mockCells,
  mockConnectedDevices,
  mockActiveSessions,
  mockRecentEvents,
  mockSessionActivity,
  mockEventsByType,
  mockIPPoolUsage,
} from './mocks/dashboardMockData';

import './App.css';

export const App: React.FC = () => {
  return (
    <div className="app-root">
      {/* Sidebar de navegação e status */}
      <Sidebar />

      {/* Área principal do Dashboard */}
      <div className="app-main">
        <Topbar />

        <main className="dashboard-container">
          {/* Linha 1: 5 KPI Cards */}
          <KpiGrid kpis={mockKpis} />

          {/* Linha 2: Seção Central (65% Topologia | 35% Sessões e Eventos) */}
          <section className="central-section">
            <div className="topology-column">
              <NetworkTopology
                cells={mockCells}
                devices={mockConnectedDevices}
              />
            </div>

            <div className="tables-column">
              <ActiveSessionsTable sessions={mockActiveSessions} />
              <RecentEventsFeed events={mockRecentEvents} />
            </div>
          </section>

          {/* Linha 3: Análises Inferiores (Session Activity, Events by Type, IP Pool) */}
          <section className="bottom-section">
            <SessionActivityChart data={mockSessionActivity} />
            <EventsByTypeChart data={mockEventsByType} />
            <IPPoolUsageCard data={mockIPPoolUsage} />
          </section>
        </main>
      </div>
    </div>
  );
};

export default App;
