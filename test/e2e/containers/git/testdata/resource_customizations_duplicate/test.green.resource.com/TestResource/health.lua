
hs = {}
if obj.status ~= nil and obj.status.color == "green" then
  hs.status = "Healthy"
  hs.message = "Healthy"
  return hs
end

hs.status = "Progressing"
hs.message = "Waiting"
return hs
