import React from 'react';
import { KpiCardProps } from '../../types';
import { KpiCard } from './KpiCard';
import './Kpi.css';

interface KpiGridProps {
  kpis: KpiCardProps[];
}

export const KpiGrid: React.FC<KpiGridProps> = ({ kpis }) => {
  return (
    <section className="kpi-grid">
      {kpis.map((kpi) => (
        <KpiCard key={kpi.id} {...kpi} />
      ))}
    </section>
  );
};
