export default function PlaceholderPage({ title }: { title: string }) {
  return (
    <div className="placeholder-page">
      <h2>{title}</h2>
      <p>Ta zakładka będzie dostępna w kolejnej wersji.</p>
    </div>
  );
}
