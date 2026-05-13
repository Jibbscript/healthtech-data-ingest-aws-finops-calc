import html2canvas from 'html2canvas';
import jsPDF from 'jspdf';

interface ExportButtonProps {
  targetId: string;
}

export function ExportButton({ targetId }: ExportButtonProps) {
  async function exportPdf() {
    const target = document.getElementById(targetId);
    if (!target) return;

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
  }

  return (
    <button className="primary" type="button" onClick={() => void exportPdf()}>
      Export PDF
    </button>
  );
}
