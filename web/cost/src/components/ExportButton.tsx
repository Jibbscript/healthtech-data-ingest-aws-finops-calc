import { useState } from 'react';

interface ExportButtonProps {
  targetId: string;
}

export function ExportButton({ targetId }: ExportButtonProps) {
  const [isExporting, setIsExporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function exportPdf() {
    setError(null);
    setIsExporting(true);
    try {
      const target = document.getElementById(targetId);
      if (!target) {
        throw new Error('Cost dashboard is unavailable.');
      }

      const [{ default: html2canvas }, { default: jsPDF }] = await Promise.all([import('html2canvas'), import('jspdf')]);
      const canvas = await html2canvas(target, { scale: 2, backgroundColor: '#f8fafc' });
      const image = canvas.toDataURL('image/png');
      const pdf = new jsPDF({ orientation: 'landscape', unit: 'pt', format: 'a4' });
      const pageWidth = pdf.internal.pageSize.getWidth();
      const pageHeight = pdf.internal.pageSize.getHeight();
      const ratio = Math.min(pageWidth / canvas.width, pageHeight / canvas.height);
      const width = canvas.width * ratio;
      const height = canvas.height * ratio;
      pdf.addImage(image, 'PNG', (pageWidth - width) / 2, 24, width, height);
      pdf.save(`throne-cost-${new Date().toISOString().replace(/[:.]/g, '-')}.pdf`);
    } catch (caught) {
      setError(`PDF export failed: ${caught instanceof Error ? caught.message : String(caught)}`);
    } finally {
      setIsExporting(false);
    }
  }

  return (
    <div className="export-control">
      <button className="primary" type="button" onClick={() => void exportPdf()} disabled={isExporting} aria-busy={isExporting}>
        {isExporting ? 'Exporting PDF' : 'Export PDF'}
      </button>
      {error ? <p className="export-error" role="alert">{error}</p> : null}
    </div>
  );
}
