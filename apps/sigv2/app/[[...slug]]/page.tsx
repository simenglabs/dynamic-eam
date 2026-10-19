import { ZonePage } from "@eam/ui";

export default function Page(props: { params: Promise<{ slug?: string[] }> }) {
  return <ZonePage zone="sigv2" params={props.params} />;
}
